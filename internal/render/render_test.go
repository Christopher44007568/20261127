package render

import (
	"encoding/xml"
	"io"
	"strings"
	"testing"

	"bipedblend/internal/biped"
)

func TestGeneratedSVGsAreWellFormed(t *testing.T) {
	result := biped.RunScenario(biped.Scenario{Duration: 2})
	strips := []biped.FilmStrip{
		biped.BuildFilmStrip(0, 4, 2),
		biped.BuildFilmStrip(1.5, 4, 2),
		biped.BuildFilmStrip(4.2, 4, 2),
	}
	docs := map[string]string{
		"frames":  FilmstripSVG(strips),
		"weights": WeightChartSVG(result),
		"slip":    SlipChartSVG(result),
	}
	for name, doc := range docs {
		decoder := xml.NewDecoder(strings.NewReader(doc))
		for {
			_, err := decoder.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("%s SVG is not well-formed: %v", name, err)
			}
		}
	}

	html := HTMLReport(result, docs["frames"], docs["weights"], docs["slip"])
	if !strings.Contains(html, "合并事件帧明细") || !strings.Contains(html, "<svg") {
		t.Fatal("HTML report is missing embedded SVG or event table")
	}
}
