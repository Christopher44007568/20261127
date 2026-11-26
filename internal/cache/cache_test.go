package cache

import (
	"strings"
	"testing"

	"shadowcache/internal/render"
)

func TestVersionedKeySeparatesShadowModes(t *testing.T) {
	c := New(DefaultOptions())
	hard := render.DefaultSettings(render.ShadowHard)
	soft := render.DefaultSettings(render.ShadowSoft)

	if _, err := c.Render(hard); err != nil {
		t.Fatal(err)
	}
	res, err := c.Render(soft)
	if err != nil {
		t.Fatal(err)
	}
	if res.Hit {
		t.Fatal("soft request hit the hard-mode entry")
	}
	if c.KeyFor(hard).String() == c.KeyFor(soft).String() {
		t.Fatal("hard and soft cache keys are identical")
	}
	if !res.Diff.Equal() {
		t.Fatalf("soft miss did not match the reference: %s", res.Diff)
	}
}

func TestHitContainsPixelEvidence(t *testing.T) {
	c := New(DefaultOptions())
	soft := render.DefaultSettings(render.ShadowSoft)

	if _, err := c.Render(soft); err != nil {
		t.Fatal(err)
	}
	res, err := c.Render(soft)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Hit {
		t.Fatal("second soft request was not a cache hit")
	}
	if !res.Diff.Equal() {
		t.Fatalf("cache hit differs from uncached reference: %s", res.Diff)
	}
	found := false
	for _, ev := range res.Events {
		if ev.Kind == "hit" {
			found = true
			if !ev.PixelsCompared || !ev.PixelsMatch {
				t.Fatalf("hit record lacks pixel validation: %+v", ev)
			}
			if ev.CompileMicros < 0 {
				t.Fatalf("invalid compile timing: %+v", ev)
			}
		}
	}
	if !found {
		t.Fatal("no hit event recorded")
	}
}

func TestPoisonedHitIsEvictedAndRebuilt(t *testing.T) {
	c := New(DefaultOptions())
	soft := render.DefaultSettings(render.ShadowSoft)

	if _, err := c.Render(soft); err != nil {
		t.Fatal(err)
	}
	key := c.KeyFor(soft)
	entry := c.entries[key]
	entry.Program.Penumbra = 0.006 // still says "soft", but renders hard edges
	c.entries[key] = entry

	res, err := c.Render(soft)
	if err != nil {
		t.Fatal(err)
	}
	if res.Hit {
		t.Fatal("poisoned entry was accepted as a valid hit")
	}
	if !res.Invalidated || !res.Rebuilt {
		t.Fatalf("poisoned entry was not evicted/rebuilt: invalidated=%t rebuilt=%t", res.Invalidated, res.Rebuilt)
	}
	if !res.Diff.Equal() {
		t.Fatalf("rebuilt frame does not match reference: %s", res.Diff)
	}
	var sawMismatch, sawRebuild bool
	for _, ev := range res.Events {
		switch ev.Kind {
		case "pixel_mismatch":
			sawMismatch = true
		case "rebuild":
			sawRebuild = true
		}
	}
	if !sawMismatch || !sawRebuild {
		t.Fatalf("missing mismatch/rebuild records: %+v", res.Events)
	}
}

func TestSchemaBumpPurgesAndRebuilds(t *testing.T) {
	c := New(DefaultOptions())
	hard := render.DefaultSettings(render.ShadowHard)
	soft := render.DefaultSettings(render.ShadowSoft)

	if _, err := c.Render(hard); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Render(soft); err != nil {
		t.Fatal(err)
	}
	if got := c.Stats().Entries; got != 2 {
		t.Fatalf("entries before bump = %d, want 2", got)
	}
	if removed := c.BumpSchema(); removed != 2 {
		t.Fatalf("removed = %d, want 2", removed)
	}
	if got := c.Stats().Entries; got != 0 {
		t.Fatalf("entries after bump = %d, want 0", got)
	}
	res, err := c.Render(soft)
	if err != nil {
		t.Fatal(err)
	}
	if res.Hit || !res.Rebuilt {
		t.Fatalf("post-bump render should rebuild: hit=%t rebuilt=%t", res.Hit, res.Rebuilt)
	}
	if !res.Diff.Equal() {
		t.Fatalf("rebuilt frame mismatch: %s", res.Diff)
	}
}

func TestShaderVersionChangePurges(t *testing.T) {
	c := New(DefaultOptions())
	soft := render.DefaultSettings(render.ShadowSoft)
	if _, err := c.Render(soft); err != nil {
		t.Fatal(err)
	}
	if removed := c.SetShaderVersion("next-shader"); removed != 1 {
		t.Fatalf("removed = %d, want 1", removed)
	}
	if got := c.Stats().Entries; got != 0 {
		t.Fatalf("entries after shader change = %d, want 0", got)
	}
}

func TestLegacyKeyReproducesWrongShader(t *testing.T) {
	c := NewLegacy()
	hard := render.DefaultSettings(render.ShadowHard)
	soft := render.DefaultSettings(render.ShadowSoft)

	if _, err := c.Render(hard); err != nil {
		t.Fatal(err)
	}
	res, err := c.Render(soft)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Hit {
		t.Fatal("legacy soft request did not hit")
	}
	if !res.WrongShader || res.Program.Mode != render.ShadowHard {
		t.Fatalf("legacy cache returned %s for soft request", res.Program.Mode)
	}
	if res.Diff.Equal() {
		t.Fatal("wrong shader unexpectedly matched the soft reference")
	}
	if strings.Contains(c.KeyFor(soft).String(), "shadow=") {
		t.Fatal("legacy key unexpectedly contains a shadow mode")
	}
}
