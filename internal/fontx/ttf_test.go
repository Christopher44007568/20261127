package fontx

import (
	"os"
	"testing"
)

func TestLoadSystemDejaVuGlyph(t *testing.T) {
	path := "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf"
	if _, err := os.Stat(path); err != nil {
		t.Skip("system DejaVu font is not available")
	}
	f, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if f.GlyphID('A') == 0 {
		t.Fatal("missing A glyph mapping")
	}
	if f.AdvanceUnits('A') <= 0 {
		t.Fatal("missing advance width")
	}
	outline, err := f.GlyphOutline('A')
	if err != nil {
		t.Fatal(err)
	}
	if len(outline.Contours) < 2 {
		t.Fatalf("A should have outer contour and counter, got %d", len(outline.Contours))
	}
}
