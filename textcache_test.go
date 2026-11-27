package main

import "testing"

func TestCachedTextShrinkLeavesNoRemnants(t *testing.T) {
	font := DefaultFont()
	c := NewCachedText(font, 3, Color{R: 255, G: 255, B: 255})

	long := "WWWWWWWWW"
	c.SetText(long)
	longW, longH := c.Size()
	if longW <= 0 || longH != 7*3 {
		t.Fatalf("unexpected long text size %dx%d", longW, longH)
	}
	surfW, surfH := c.surf.W, c.surf.H
	if surfW < longW {
		t.Fatalf("cache surface width %d smaller than text %d", surfW, longW)
	}

	// Record that the tail columns really contained opaque glyph texels.
	foundOpaque := false
	for y := 0; y < surfH; y++ {
		if c.SurfaceAlpha(surfW-2, y) == 255 {
			foundOpaque = true
		}
	}
	if !foundOpaque {
		t.Fatal("test setup: expected opaque texels near right edge of long string")
	}

	c.SetText("I")
	shortW, shortH := c.Size()
	if shortW >= longW {
		t.Fatalf("short text width %d should be less than long width %d", shortW, longW)
	}

	// Every storage texel outside the tight new text must be transparent:
	// this is the direct guard against "previous string leftovers".
	for y := 0; y < surfH; y++ {
		for x := 0; x < surfW; x++ {
			a := c.SurfaceAlpha(x, y)
			if x >= shortW && a != 0 {
				t.Fatalf("stale glyph texel at storage (%d,%d) alpha=%d beyond new width %d", x, y, a, shortW)
			}
			if y >= shortH && a != 0 {
				t.Fatalf("stale glyph texel at storage (%d,%d) alpha=%d beyond new height %d", x, y, a, shortH)
			}
		}
	}

	// The tight view of the cache surface must equal a freshly rendered "I".
	fresh := font.Render("I", 3, Color{R: 255, G: 255, B: 255})
	got := c.Surface().View(shortW, shortH)
	if got.W != fresh.W || got.H != fresh.H {
		t.Fatalf("sprite size %dx%d, want %dx%d", got.W, got.H, fresh.W, fresh.H)
	}
	for y := 0; y < fresh.H; y++ {
		for x := 0; x < fresh.W; x++ {
			gi := 4 * (y*got.Stride + x)
			fi := 4 * (y*fresh.Stride + x)
			if got.Pix[gi] != fresh.Pix[fi] || got.Pix[gi+1] != fresh.Pix[fi+1] ||
				got.Pix[gi+2] != fresh.Pix[fi+2] || got.Pix[gi+3] != fresh.Pix[fi+3] {
				t.Fatalf("cache pixel (%d,%d) differs from fresh render", x, y)
			}
		}
	}
}

func TestCachedTextSameStringNoRebuild(t *testing.T) {
	c := NewCachedText(DefaultFont(), 2, Color{R: 1, G: 2, B: 3})
	c.SetText("ABC")
	v1, r1 := c.Version(), c.Rebuilds()
	if c.SetText("ABC") {
		t.Fatal("SetText reported change for identical string")
	}
	if c.Version() != v1 || c.Rebuilds() != r1 {
		t.Fatal("cache rebuilt despite identical string")
	}
	if !c.SetText("ABCD") {
		t.Fatal("SetText did not report change for different string")
	}
	if c.Version() != v1+1 || c.Rebuilds() != r1+1 {
		t.Fatal("version/rebuild counters did not advance")
	}
}

func TestFontMeasureMatchesRender(t *testing.T) {
	f := DefaultFont()
	for _, s := range []string{"", "A", "HELLO", "FRAME 03/24", "aZ 09!", "UNKNOWN~\x80"} {
		for _, scale := range []int{1, 2, 4} {
			w, h := f.Measure(s, scale)
			sp := f.Render(s, scale, Color{R: 255, G: 255, B: 255})
			if s == "" {
				if sp.W != 1 || sp.H != h {
					t.Fatalf("empty render size %dx%d, want 1x%d", sp.W, sp.H, h)
				}
				continue
			}
			if sp.W != w || sp.H != h {
				t.Fatalf("Render(%q,%d) size %dx%d, measure %dx%d", s, scale, sp.W, sp.H, w, h)
			}
		}
	}
}
