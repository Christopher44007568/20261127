package render

import "testing"

// TestGlyphCoverage guarantees every character used by the figure labels
// has an embedded bitmap, so titles cannot silently drop letters.
func TestGlyphCoverage(t *testing.T) {
	labels := []string{
		"THIN FILM COLOR: THICKNESS AND ANGLE",
		"REFLECTANCE SPECTRA (NORMAL INCIDENCE)",
		"BARE METAL VS FILM-COATED SPHERE",
		"FILM THICKNESS (NM)",
		"INCIDENCE ANGLE (DEG)",
		"WAVELENGTH (NM)",
		"REFLECTANCE",
		"D=0NM (BARE)",
		"BARE IRON (D=0)",
		"COATED IRON D=500NM",
		"0123456789 .,:=+-/%()",
	}
	for _, s := range labels {
		for i := 0; i < len(s); i++ {
			if _, ok := glyph5x7[s[i]]; !ok {
				t.Fatalf("missing glyph for %q in %q", s[i], s)
			}
		}
	}
}
