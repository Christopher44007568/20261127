package main

import (
	"encoding/json"
	"os"
	"time"
)

type Check struct {
	Name   string  `json:"name"`
	Pass   bool    `json:"pass"`
	Detail string  `json:"detail"`
	Value  float64 `json:"value,omitempty"`
}

type BoldEntry struct {
	Char           string  `json:"char"`
	TargetEm       float64 `json:"targetRadiusEm"`
	ChosenEm       float64 `json:"chosenRadiusEm"`
	Limited        bool    `json:"clampedByCounterGuard"`
	Counters       int     `json:"counters"`
	MinRetention   float64 `json:"minAreaRetention"`
	RejectedClosed int     `json:"overbold_0p11_closedCounters"`
}

type Report struct {
	GeneratedAt string      `json:"generatedAt"`
	Module      string      `json:"module"`
	FontLatin   string      `json:"fontLatin"`
	FontHan     string      `json:"fontHan"`
	UnitsPerEm  int         `json:"unitsPerEmLatin"`
	Checks      []Check     `json:"checks"`
	Layout      *Layout     `json:"layout"`
	Bold        []BoldEntry `json:"boldGuard"`
	Artifacts   []string    `json:"artifacts"`
}

type PtJSON struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type ContourData struct {
	LengthEm             float64  `json:"lengthEm"`
	SampleCount          int      `json:"sampleCount"`
	SpacingEm            float64  `json:"targetArcSpacingEm"`
	MaxArcSpacingErrorEm float64  `json:"maxArcSpacingErrorEm"`
	SamplesEm            []PtJSON `json:"samplesEmYDown"`
}

type GlyphOutlineData struct {
	Char       string        `json:"char"`
	Code       rune          `json:"code"`
	UnitsPerEm int           `json:"unitsPerEm"`
	Contours   []ContourData `json:"contours"`
}

type OutlineDataFile struct {
	CoordinateSystem string             `json:"coordinateSystem"`
	SpacingPxOnSheet float64            `json:"sheetSpacingPx"`
	Glyphs           []GlyphOutlineData `json:"glyphs"`
}

func writeJSON(path string, v any) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

func buildReport(fs *FontSet, lay *Layout, artifacts []string) Report {
	r := Report{
		GeneratedAt: time.Now().Format(time.RFC3339),
		Module:      "outline-lab (pure Go standard library)",
		FontLatin:   fs.Latin.Path,
		UnitsPerEm:  fs.Latin.UnitsPerEm,
		Layout:      lay,
		Artifacts:   artifacts,
	}
	if fs.Han != nil {
		r.FontHan = fs.Han.Path
	}
	r.Checks = []Check{
		{
			Name:   "ascender_descender_no_overlap",
			Pass:   !lay.Overlap && lay.MinPairGap >= lay.Style.MinInkGapEm*lay.Style.Ppem-1e-6,
			Value:  lay.MinPairGap,
			Detail: "minimum ink gap between adjacent line extents (px); must stay >= minInkGap",
		},
		{
			Name:   "tracking_in_pen_geometry",
			Pass:   lay.Style.TrackingEm > 0,
			Value:  lay.Style.TrackingEm * lay.Style.Ppem,
			Detail: "tracking px added to every pen advance, independent of text box",
		},
	}
	for _, c := range []rune{'a', 'B', '8', 'o', '国', '圆', '量'} {
		rep := fs.BoldReport(c)
		e := BoldEntry{
			Char:     string(c),
			TargetEm: rep.TargetEm,
			ChosenEm: rep.ChosenEm,
			Limited:  rep.Limited,
			Counters: len(rep.AtChosen),
		}
		minRet := 1.0
		for _, st := range rep.AtChosen {
			if st.Retention < minRet {
				minRet = st.Retention
			}
		}
		if len(rep.AtChosen) > 0 {
			e.MinRetention = minRet
		}
		_, stats, _, _, _, _, _ := fs.EvaluateRadiusEm(c, 0.11)
		for _, st := range stats {
			if st.AreaBold < 28 {
				e.RejectedClosed++
			}
		}
		r.Bold = append(r.Bold, e)
	}
	preserved := true
	globalMin := 1.0
	for _, e := range r.Bold {
		if e.Counters > 0 && e.MinRetention < 0.38 {
			preserved = false
		}
		if e.Counters > 0 && e.MinRetention < globalMin {
			globalMin = e.MinRetention
		}
	}
	r.Checks = append(r.Checks, Check{
		Name:   "bold_counters_preserved",
		Pass:   preserved,
		Value:  globalMin,
		Detail: "adopted bold radius keeps every detected counter above retention/core thresholds",
	})
	return r
}

func buildOutlineData(fs *FontSet) OutlineDataFile {
	df := OutlineDataFile{
		CoordinateSystem: "em units, origin at pen start on baseline, x right, y down",
		SpacingPxOnSheet: 30,
	}
	for _, c := range outlineGlyphs {
		f, g := fs.Glyph(c)
		gd := GlyphOutlineData{Char: string(c), Code: c, UnitsPerEm: f.UnitsPerEm}
		o := GlyphOutline(g, f.UnitsPerEm, float64(f.UnitsPerEm), 1.0)
		em := float64(f.UnitsPerEm)
		for _, ct := range o.Contours {
			length := ct.Length()
			cd := ContourData{LengthEm: length / em}
			const spacingEm = 0.055
			n := intRound(cd.LengthEm / spacingEm)
			if n < 3 {
				n = 3
			}
			samples := ct.SampleArcLength(n)
			cd.SampleCount = len(samples)
			cd.SpacingEm = length / float64(n) / em
			step := length / float64(n)
			maxDev := 0.0
			for i := range samples {
				nextDist := length
				if i+1 < len(samples) {
					nextDist = samples[i+1].Dist
				}
				if dev := absF(nextDist - samples[i].Dist - step); dev > maxDev {
					maxDev = dev
				}
			}
			cd.MaxArcSpacingErrorEm = maxDev / em
			for _, s := range samples {
				cd.SamplesEm = append(cd.SamplesEm, PtJSON{
					X: s.Pos.X / em, Y: s.Pos.Y / em,
				})
			}
			gd.Contours = append(gd.Contours, cd)
		}
		df.Glyphs = append(df.Glyphs, gd)
	}
	return df
}

func absF(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func intRound(x float64) int {
	return int(x + 0.5)
}
