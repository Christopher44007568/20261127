package cache

import (
	"fmt"
	"time"

	"shadowcache/internal/render"
)

type Entry struct {
	Key          Key
	Program      render.Program
	RenderedHash string
}

type Options struct {
	SchemaVersion uint32
	ShaderVersion string
	SceneVersion  string
}

func DefaultOptions() Options {
	return Options{
		SchemaVersion: 3,
		ShaderVersion: render.ShaderVersion,
		SceneVersion:  render.SceneVersion,
	}
}

type Cache struct {
	schema        uint32
	shaderVersion string
	sceneVersion  string
	entries       map[Key]Entry
	events        []Event
	nextSeq       int
	hits          int
	misses        int
	invalidations int
	dirty         bool
}

func New(opts Options) *Cache {
	if opts.SchemaVersion == 0 {
		opts.SchemaVersion = 1
	}
	if opts.ShaderVersion == "" {
		opts.ShaderVersion = render.ShaderVersion
	}
	if opts.SceneVersion == "" {
		opts.SceneVersion = render.SceneVersion
	}
	return &Cache{
		schema:        opts.SchemaVersion,
		shaderVersion: opts.ShaderVersion,
		sceneVersion:  opts.SceneVersion,
		entries:       make(map[Key]Entry),
	}
}

func (c *Cache) KeyFor(s render.Settings) Key {
	return Key{
		Schema:        c.schema,
		ShaderVersion: c.shaderVersion,
		SceneVersion:  s.SceneVersion,
		Width:         s.Width,
		Height:        s.Height,
		ShadowMode:    s.ShadowMode,
	}
}

type Result struct {
	Key          Key
	Frame        *render.Frame
	Reference    *render.Frame
	Program      render.Program
	Hit          bool
	Rebuilt      bool
	Invalidated  bool
	WrongShader  bool
	Diff         render.Diff
	Events       []Event
	LookupMicros int64
}

func (c *Cache) Render(s render.Settings) (Result, error) {
	if err := s.Validate(); err != nil {
		return Result{}, err
	}
	if s.SceneVersion != c.sceneVersion {
		c.sceneVersion = s.SceneVersion
		c.PurgeStale("scene version changed to " + s.SceneVersion)
	}
	key := c.KeyFor(s)
	result := Result{Key: key}

	lookupStart := time.Now()
	entry, hit := c.entries[key]
	result.LookupMicros = time.Since(lookupStart).Microseconds()
	if hit && !c.entryCurrent(entry) {
		delete(c.entries, key)
		c.invalidations++
		c.dirty = true
		ev := c.newEvent("invalidate", key, s.ShadowMode)
		ev.Detail = "entry failed version/program invariant check"
		c.events = append(c.events, ev)
		result.Events = append(result.Events, ev)
		result.Invalidated = true
		hit = false
	}

	if hit {
		cachedFrame, err := render.Render(s, entry.Program)
		if err != nil {
			return Result{}, err
		}
		ref, _, err := render.RenderDirect(s)
		if err != nil {
			return Result{}, err
		}
		diff := render.Compare(cachedFrame, ref)
		ev := c.newEvent("hit", key, s.ShadowMode)
		ev.ReturnedShadow = entry.Program.Mode.String()
		ev.ProgramID = entry.Program.ID
		ev.CompileMicros = entry.Program.CompileMicros
		ev.PixelsCompared = true
		ev.PixelsMatch = diff.Equal()
		ev.PixelHash = cachedFrame.Hash()
		ev.ReferenceHash = ref.Hash()
		ev.DifferentPixels = diff.DifferentPixels
		ev.Detail = "key matched; pixels compared with uncached reference"
		c.events = append(c.events, ev)
		result.Events = append(result.Events, ev)

		if diff.Equal() {
			entry.RenderedHash = ev.PixelHash
			c.entries[key] = entry
			c.hits++
			result.Frame = cachedFrame
			result.Reference = ref
			result.Program = entry.Program
			result.Hit = true
			result.Diff = diff
			return result, nil
		}

		// A key can match while the stored program is still wrong (corruption,
		// a failed migration, or a legacy key writer). Pixel evidence wins.
		delete(c.entries, key)
		c.invalidations++
		c.dirty = true
		ev = c.newEvent("pixel_mismatch", key, s.ShadowMode)
		ev.ReturnedShadow = entry.Program.Mode.String()
		ev.ProgramID = entry.Program.ID
		ev.PixelsCompared = true
		ev.PixelsMatch = false
		ev.PixelHash = cachedFrame.Hash()
		ev.ReferenceHash = ref.Hash()
		ev.DifferentPixels = diff.DifferentPixels
		ev.Detail = "evicted cached entry because pixels differed from uncached reference"
		c.events = append(c.events, ev)
		result.Events = append(result.Events, ev)
		result.Invalidated = true
	}

	compileStart := time.Now()
	program := render.CompileShader(s.ShadowMode)
	compileMicros := time.Since(compileStart).Microseconds()
	frame, err := render.Render(s, program)
	if err != nil {
		return Result{}, err
	}
	ref, _, err := render.RenderDirect(s)
	if err != nil {
		return Result{}, err
	}
	diff := render.Compare(frame, ref)
	if !diff.Equal() {
		return Result{}, fmt.Errorf("fresh shader did not match uncached reference: %s", diff)
	}

	ev := c.newEvent("miss", key, s.ShadowMode)
	rebuilt := result.Invalidated || c.dirty
	if rebuilt {
		ev.Kind = "rebuild"
	}
	ev.ReturnedShadow = program.Mode.String()
	ev.ProgramID = program.ID
	ev.CompileMicros = compileMicros
	ev.PixelsCompared = true
	ev.PixelsMatch = true
	ev.PixelHash = frame.Hash()
	ev.ReferenceHash = ref.Hash()
	ev.Detail = "compiled requested shadow variant and verified against uncached reference"
	c.events = append(c.events, ev)
	result.Events = append(result.Events, ev)

	c.entries[key] = Entry{Key: key, Program: program, RenderedHash: ev.PixelHash}
	c.misses++
	c.dirty = false
	result.Frame = frame
	result.Reference = ref
	result.Program = program
	result.Rebuilt = rebuilt
	result.Diff = diff
	return result, nil
}

func (c *Cache) entryCurrent(entry Entry) bool {
	return entry.Key.Schema == c.schema &&
		entry.Key.ShaderVersion == c.shaderVersion &&
		entry.Key.SceneVersion == c.sceneVersion &&
		entry.Program.ShaderVersion == c.shaderVersion &&
		entry.Program.Mode == entry.Key.ShadowMode
}

func (c *Cache) newEvent(kind string, key Key, mode render.ShadowMode) Event {
	c.nextSeq++
	return Event{
		Seq:             c.nextSeq,
		Kind:            kind,
		Key:             key.String(),
		KeyDigest:       key.Digest(),
		RequestedShadow: mode.String(),
		SchemaVersion:   c.schema,
	}
}

func (c *Cache) Stats() Stats {
	return Stats{
		Hits:          c.hits,
		Misses:        c.misses,
		Invalidations: c.invalidations,
		Entries:       len(c.entries),
		SchemaVersion: c.schema,
	}
}

func (c *Cache) Events() []Event {
	out := make([]Event, len(c.events))
	copy(out, c.events)
	return out
}

// BumpSchema invalidates every entry by moving to a new cache-key schema. The
// old entries are physically purged so a later lookup cannot see them.
func (c *Cache) BumpSchema() int {
	c.schema++
	return c.PurgeStale(fmt.Sprintf("cache schema bumped to v%d", c.schema))
}

// SetShaderVersion models a hot reload of shader source/semantics.
func (c *Cache) SetShaderVersion(version string) int {
	c.shaderVersion = version
	return c.PurgeStale("shader version changed to " + version)
}

// SetSceneVersion models a scene-definition change.
func (c *Cache) SetSceneVersion(version string) int {
	c.sceneVersion = version
	return c.PurgeStale("scene version changed to " + version)
}

func (c *Cache) InvalidateMode(mode render.ShadowMode) int {
	removed := 0
	for key := range c.entries {
		if key.ShadowMode == mode {
			delete(c.entries, key)
			removed++
		}
	}
	c.recordInvalidations(removed, "invalidated shadow mode "+mode.String())
	return removed
}

func (c *Cache) InvalidateAll() int {
	removed := len(c.entries)
	clear(c.entries)
	c.recordInvalidations(removed, "invalidated all entries")
	return removed
}

func (c *Cache) PurgeStale(reason string) int {
	removed := 0
	for key, entry := range c.entries {
		if key.Schema != c.schema ||
			key.ShaderVersion != c.shaderVersion ||
			key.SceneVersion != c.sceneVersion ||
			entry.Program.ShaderVersion != c.shaderVersion ||
			entry.Program.Mode != key.ShadowMode {
			delete(c.entries, key)
			removed++
		}
	}
	c.recordInvalidations(removed, reason)
	return removed
}

func (c *Cache) recordInvalidations(removed int, reason string) {
	if removed == 0 {
		return
	}
	c.invalidations += removed
	c.dirty = true
	c.nextSeq++
	c.events = append(c.events, Event{
		Seq:           c.nextSeq,
		Kind:          "invalidate",
		SchemaVersion: c.schema,
		Detail:        fmt.Sprintf("%s; removed %d entries", reason, removed),
	})
}
