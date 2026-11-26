package main

import (
	"errors"
	"fmt"
)

type ResourceState uint8

const (
	ResourceLive ResourceState = iota
	ResourceRetired
	ResourceReclaimed
)

func (s ResourceState) String() string {
	switch s {
	case ResourceLive:
		return "live"
	case ResourceRetired:
		return "retired"
	case ResourceReclaimed:
		return "reclaimed"
	default:
		return "unknown"
	}
}

type ReclaimPolicy struct {
	// DeferredRelease is the safe policy: zero-referenced resources enter the
	// queue and are reclaimed only after a frame boundary.
	DeferredRelease bool
	// ExtraDelayFrames deliberately keeps retired resources alive for extra
	// frames. It exists only to reproduce late-reclaim ghosting.
	ExtraDelayFrames int
}

var (
	PolicyCorrect = ReclaimPolicy{DeferredRelease: true}
	PolicyEarly   = ReclaimPolicy{DeferredRelease: false}
	PolicyLate    = ReclaimPolicy{DeferredRelease: true, ExtraDelayFrames: 1}
)

// Resource owns an offscreen surface. A resource may not be acquired again
// after its final lease is released, even if the deferred queue still owns the
// backing allocation.
type Resource struct {
	ID           string
	Generation   uint64
	Surface      *Surface
	Refs         int
	State        ResourceState
	CreatedFrame int
	RetiredFrame int
}

// Lease is one pass's reference to a resource. Release is explicit so the
// producer and each consumer can independently keep the texture alive.
type Lease struct {
	id       string
	gen      uint64
	pass     string
	released bool
}

func (l *Lease) ID() string         { return l.id }
func (l *Lease) Generation() uint64 { return l.gen }
func (l *Lease) Pass() string       { return l.pass }

type reclaimEntry struct {
	id            string
	gen           uint64
	releaseFrame  int
	eligibleFrame int
}

type LifecycleStats struct {
	Acquires       int
	FailedAcquires int
	Releases       int
	DoubleReleases int
	InvalidSamples int
	Retired        int
	Reclaimed      int
	ReusedRetired  int
	QueueHighWater int
}

type Manager struct {
	policy  ReclaimPolicy
	frame   int
	started bool
	nextGen uint64
	state   map[string]*Resource
	queue   []reclaimEntry
	Stats   LifecycleStats
}

func NewManager(policy ReclaimPolicy) *Manager {
	return &Manager{
		policy: policy,
		frame:  -1,
		state:  make(map[string]*Resource),
	}
}

func (m *Manager) Policy() ReclaimPolicy { return m.policy }
func (m *Manager) Frame() int            { return m.frame }

// BeginFrame closes the previous frame boundary before any pass in this frame
// runs. It is the only place where queued resources may be physically dropped.
func (m *Manager) BeginFrame(frame int) error {
	if frame < 0 {
		return fmt.Errorf("negative frame %d", frame)
	}
	if m.started && frame <= m.frame {
		return fmt.Errorf("frame %d must follow frame %d", frame, m.frame)
	}
	m.frame = frame
	m.started = true
	m.reclaimEligible()
	return nil
}

func (m *Manager) EndFrame() error {
	if !m.started {
		return errors.New("EndFrame called before BeginFrame")
	}
	return nil
}

func (m *Manager) Create(id string, w, h int) (*Resource, error) {
	if !m.started {
		return nil, errors.New("resource creation outside a frame")
	}
	if _, exists := m.state[id]; exists {
		return nil, fmt.Errorf("resource %q already exists", id)
	}
	m.nextGen++
	r := &Resource{
		ID:           id,
		Generation:   m.nextGen,
		Surface:      NewSurface(w, h),
		State:        ResourceLive,
		CreatedFrame: m.frame,
		RetiredFrame: -1,
	}
	m.state[id] = r
	return r, nil
}

func (m *Manager) Acquire(id, pass string) (*Lease, error) {
	r := m.state[id]
	if r == nil || r.State != ResourceLive {
		m.Stats.FailedAcquires++
		return nil, fmt.Errorf("acquire %q for pass %q: resource is not live", id, pass)
	}
	r.Refs++
	m.Stats.Acquires++
	return &Lease{id: id, gen: r.Generation, pass: pass}, nil
}

func (m *Manager) Release(l *Lease) error {
	if l == nil {
		m.Stats.DoubleReleases++
		return errors.New("nil lease")
	}
	if l.released {
		m.Stats.DoubleReleases++
		return fmt.Errorf("lease %q for pass %q released twice", l.id, l.pass)
	}
	l.released = true
	r := m.state[l.id]
	if r == nil || r.Generation != l.gen {
		m.Stats.DoubleReleases++
		return fmt.Errorf("lease %q generation %d is stale", l.id, l.gen)
	}
	if r.Refs <= 0 {
		m.Stats.DoubleReleases++
		return fmt.Errorf("resource %q has no live references", l.id)
	}
	r.Refs--
	m.Stats.Releases++
	if r.Refs > 0 {
		return nil
	}

	r.State = ResourceRetired
	r.RetiredFrame = m.frame
	m.Stats.Retired++
	if !m.policy.DeferredRelease {
		m.destroy(r)
		return nil
	}

	m.queue = append(m.queue, reclaimEntry{
		id:            r.ID,
		gen:           r.Generation,
		releaseFrame:  m.frame,
		eligibleFrame: m.frame + 1 + m.policy.ExtraDelayFrames,
	})
	m.Stats.QueueHighWater = maxInt(m.Stats.QueueHighWater, len(m.queue))
	return nil
}

func (m *Manager) Surface(l *Lease) (*Surface, error) {
	if l == nil || l.released {
		return nil, fmt.Errorf("use of released lease")
	}
	r := m.state[l.id]
	if r == nil || r.Generation != l.gen || r.State != ResourceLive || r.Refs <= 0 {
		return nil, fmt.Errorf("resource %q is unavailable", l.id)
	}
	return r.Surface, nil
}

func (m *Manager) Sample(l *Lease, x, y int) (RGBA, error) {
	s, err := m.Surface(l)
	if err != nil {
		m.Stats.InvalidSamples++
		return colTransparent, err
	}
	return s.At(x, y), nil
}

// ReuseRetired reproduces a late-cache bug: it revives a retired allocation
// without clearing its previous-frame pixels. The renderer uses this only for
// the delayed-reclaim diagnostic.
func (m *Manager) ReuseRetired(id string) (*Resource, error) {
	r := m.state[id]
	if r == nil || r.State != ResourceRetired {
		return nil, fmt.Errorf("retired resource %q is not available for reuse", id)
	}
	for i := range m.queue {
		if m.queue[i].id == id && m.queue[i].gen == r.Generation {
			m.queue = append(m.queue[:i], m.queue[i+1:]...)
			break
		}
	}
	r.State = ResourceLive
	m.Stats.ReusedRetired++
	return r, nil
}

func (m *Manager) Info(id string) (ResourceState, int, uint64, bool) {
	r, ok := m.state[id]
	if !ok {
		return ResourceReclaimed, 0, 0, false
	}
	return r.State, r.Refs, r.Generation, true
}

func (m *Manager) QueueLen() int { return len(m.queue) }

func (m *Manager) reclaimEligible() {
	kept := m.queue[:0]
	for _, q := range m.queue {
		if q.eligibleFrame <= m.frame {
			if r := m.state[q.id]; r != nil && r.Generation == q.gen {
				m.destroy(r)
			}
			continue
		}
		kept = append(kept, q)
	}
	m.queue = kept
}

func (m *Manager) destroy(r *Resource) {
	delete(m.state, r.ID)
	r.State = ResourceReclaimed
	r.Refs = 0
	r.Surface = nil
	m.Stats.Reclaimed++
}
