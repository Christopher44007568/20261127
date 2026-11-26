package demo

import (
	"encoding/json"
	"image"
	"os"
)

// Check is one machine-verifiable acceptance item.
type Check struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

// MaskStat summarizes one output mask.
type MaskStat struct {
	File         string `json:"file"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	Covered      int    `json:"covered_pixels"`
	BinaryOnly   bool   `json:"binary_only"`
	Intermediate bool   `json:"has_intermediate_values"`
}

// Report is written to artifacts/report.json.
type Report struct {
	OK              bool         `json:"ok"`
	Checks          []Check      `json:"checks"`
	Masks           []MaskStat   `json:"masks"`
	RuleDiffPixels  int          `json:"rule_different_pixels"`
	ProtectedHolePx int          `json:"protected_hole_pixels"`
	Seams           []SeamReport `json:"clip_seams"`
	Outputs         []string     `json:"outputs"`
}

func (r *Report) add(name string, ok bool, detail string) {
	r.Checks = append(r.Checks, Check{Name: name, OK: ok, Detail: detail})
	if !ok {
		r.OK = false
	}
}

// ExpectedSample is a point whose mask value must equal Want.
type ExpectedSample struct {
	Name string
	X, Y int
	Want byte
}

// NestedSamples encodes the nesting contract for NestedPath.
func NestedSamples() []ExpectedSample {
	return []ExpectedSample{
		{"outer_top_band", 180, 50, 255},
		{"outer_left_band", 60, 150, 255},
		{"outer_bottom_band", 180, 250, 255},
		{"hole_nw", 130, 110, 0},
		{"hole_south", 180, 200, 0},
		{"hole_east", 235, 150, 0},
		{"island_center", 180, 150, 255},
		{"outside_tl", 12, 12, 0},
		{"outside_br", 348, 288, 0},
	}
}

// VerifyMasks checks fill-rule differences, nesting and binary coverage.
func VerifyMasks(masks map[string]*Mask, rep *Report) {
	rep.OK = true

	eo := masks["mask_star_evenodd"]
	nz := masks["mask_star_nonzero"]
	diff := 0
	for i := range eo.Pix {
		if eo.Pix[i] != nz.Pix[i] {
			diff++
		}
	}
	rep.RuleDiffPixels = diff
	centerEO := eo.At(180, 152)
	centerNZ := nz.At(180, 152)
	rep.add("star_rules_produce_different_holes",
		diff > 100 && centerEO == 0 && centerNZ == 255,
		"central pixel evenodd=0 nonzero=255; differing pixels counted")

	for _, key := range []string{"mask_nested_evenodd", "mask_nested_nonzero"} {
		m := masks[key]
		allOK := true
		bad := ""
		for _, s := range NestedSamples() {
			if got := m.At(s.X, s.Y); got != s.Want {
				allOK = false
				bad = s.Name
				break
			}
		}
		detail := "all nesting samples match"
		if !allOK {
			detail = "first failing sample: " + bad
		}
		rep.add(key+"_nesting_samples", allOK, detail)
	}

	for name, m := range masks {
		partial := m.HasIntermediateValue()
		rep.add(name+"_binary_coverage", !partial, "mask contains only 0/255")
		rep.Masks = append(rep.Masks, MaskStat{
			File:         name + ".png",
			Width:        m.W,
			Height:       m.H,
			Covered:      m.Count(),
			BinaryOnly:   !partial,
			Intermediate: partial,
		})
	}
}

// VerifyHoleCompositing proves the main outer contour does not cover holes in
// the final color image: every pixel that the outer contour alone covers but
// the unified nested mask leaves open must be byte-identical to the paper
// background after painting.
func VerifyHoleCompositing(paper, composite *image.NRGBA, nested *Mask, rep *Report) {
	outer := NewMask(SceneW, SceneH)
	FillPath(outer, AddRoundedRect(&Path{}, 36, 28, 324, 272, 30, WindingClockwise), FillNonZero)

	protected := 0
	leaked := 0
	var first image.Point
	for y := 0; y < SceneH; y++ {
		for x := 0; x < SceneW; x++ {
			if outer.At(x, y) == 255 && nested.At(x, y) == 0 {
				protected++
				pi := paper.PixOffset(x, y)
				ci := composite.PixOffset(x, y)
				if paper.Pix[pi+0] != composite.Pix[ci+0] ||
					paper.Pix[pi+1] != composite.Pix[ci+1] ||
					paper.Pix[pi+2] != composite.Pix[ci+2] {
					leaked++
					if leaked == 1 {
						first = image.Pt(x, y)
					}
				}
			}
		}
	}
	rep.ProtectedHolePx = protected
	detail := "outer-only hole pixels left identical to paper; leaked=" + itoa(leaked)
	if leaked > 0 {
		detail += " first_leak=" + first.String()
	}
	rep.add("holes_not_covered_by_main_contour",
		protected > 1000 && leaked == 0, detail)
}

// VerifyClipSeams checks all four edges of ClipRectNested.
func VerifyClipSeams(before, clipped *Mask, rep *Report) {
	vL := CheckVerticalClipSeam(clipped, before, ClipRectNested.Min.X, false, 80, 224)
	vR := CheckVerticalClipSeam(clipped, before, ClipRectNested.Max.X, true, 80, 224)
	hT := CheckHorizontalClipSeam(clipped, before, ClipRectNested.Min.Y, false, 120, 250)
	hB := CheckHorizontalClipSeam(clipped, before, ClipRectNested.Max.Y, true, 120, 250)
	rep.Seams = []SeamReport{vL, vR, hT, hB}
	allOK := vL.OK && vR.OK && hT.OK && hB.OK
	rep.add("clip_edges_have_no_half_pixel_gap", allOK,
		"all four edges: full edge pixels, zero sandwiched gaps, binary coverage")
}

// VerifySnap checks outward snapping of fractional clip rectangles.
func VerifySnap(rep *Report) {
	r := SnapClipRectOutward(10.2, 20.7, 30.9, 40.1)
	ok := r == image.Rect(10, 20, 31, 41)
	rep.add("fractional_clip_snaps_outward", ok, r.String())
}

// WriteReport writes indented JSON.
func WriteReport(path string, rep *Report) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(rep)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
