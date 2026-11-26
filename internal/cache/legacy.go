package cache

import "shadowcache/internal/render"

// LegacyCache is the buggy cache from the report. Its key has no ShadowMode.
type LegacyCache struct {
	entries map[LegacyKey]render.Program
	events  []Event
	nextSeq int
	hits    int
	misses  int
}

func NewLegacy() *LegacyCache {
	return &LegacyCache{entries: make(map[LegacyKey]render.Program)}
}

func (c *LegacyCache) KeyFor(s render.Settings) LegacyKey {
	return LegacyKey{
		ShaderVersion: render.ShaderVersion,
		SceneVersion:  s.SceneVersion,
		Width:         s.Width,
		Height:        s.Height,
	}
}

func (c *LegacyCache) Render(s render.Settings) (Result, error) {
	if err := s.Validate(); err != nil {
		return Result{}, err
	}
	key := c.KeyFor(s)

	program, hit := c.entries[key]
	if hit {
		frame, err := render.Render(s, program)
		if err != nil {
			return Result{}, err
		}
		ref, _, err := render.RenderDirect(s)
		if err != nil {
			return Result{}, err
		}
		diff := render.Compare(frame, ref)
		ev := c.newEvent("hit", key, s.ShadowMode)
		ev.ReturnedShadow = program.Mode.String()
		ev.ProgramID = program.ID
		ev.CompileMicros = program.CompileMicros
		ev.PixelsCompared = true
		ev.PixelsMatch = diff.Equal()
		ev.PixelHash = frame.Hash()
		ev.ReferenceHash = ref.Hash()
		ev.DifferentPixels = diff.DifferentPixels
		ev.Detail = "legacy key lacks shadow mode; cached program reused"
		c.events = append(c.events, ev)
		c.hits++
		return Result{
			Frame:       frame,
			Reference:   ref,
			Program:     program,
			Hit:         true,
			WrongShader: program.Mode != s.ShadowMode,
			Diff:        diff,
			Events:      []Event{ev},
		}, nil
	}

	program = render.CompileShader(s.ShadowMode)
	frame, err := render.Render(s, program)
	if err != nil {
		return Result{}, err
	}
	ref, _, err := render.RenderDirect(s)
	if err != nil {
		return Result{}, err
	}
	diff := render.Compare(frame, ref)
	ev := c.newEvent("miss", key, s.ShadowMode)
	ev.ReturnedShadow = program.Mode.String()
	ev.ProgramID = program.ID
	ev.CompileMicros = program.CompileMicros
	ev.PixelsCompared = true
	ev.PixelsMatch = diff.Equal()
	ev.PixelHash = frame.Hash()
	ev.ReferenceHash = ref.Hash()
	ev.DifferentPixels = diff.DifferentPixels
	ev.Detail = "legacy key miss; compiled requested shadow variant"
	c.events = append(c.events, ev)
	c.entries[key] = program
	c.misses++
	return Result{
		Frame:     frame,
		Reference: ref,
		Program:   program,
		Diff:      diff,
		Events:    []Event{ev},
	}, nil
}

func (c *LegacyCache) newEvent(kind string, key LegacyKey, mode render.ShadowMode) Event {
	c.nextSeq++
	return Event{
		Seq:             c.nextSeq,
		Kind:            kind,
		Key:             key.String(),
		KeyDigest:       key.Digest(),
		RequestedShadow: mode.String(),
		SchemaVersion:   0,
	}
}

func (c *LegacyCache) Stats() Stats {
	return Stats{Hits: c.hits, Misses: c.misses, Entries: len(c.entries)}
}

func (c *LegacyCache) Events() []Event {
	out := make([]Event, len(c.events))
	copy(out, c.events)
	return out
}
