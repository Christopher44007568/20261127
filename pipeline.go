package main

import (
	"image"
	"path/filepath"
)

// runFaultDemo deliberately corrupts one semantic property (the radial
// gradient becomes a flat wrong fill), re-renders and verifies that the diff
// machinery attributes the damage to the original object ID.
func runFaultDemo(outDir string, ref *image.RGBA, scene *Scene, ss int, threshold float64) *FaultInfo {
	var target *Obj
	for _, o := range scene.All {
		if o.ID == "obj-radial-gradient" {
			target = o
		}
	}
	if target == nil {
		return nil
	}
	originalFill := target.Fill
	target.Fill = solid("#64748b")

	fss := ss
	if fss > 3 {
		fss = 3
	}
	rr := Render(scene, fss)
	diff, rep := ComputeDiff(ref, rr, threshold)

	topIdx, topErr := -1, -1
	for i, st := range rep.Objects {
		if st.ErrorPixels > topErr {
			topErr = st.ErrorPixels
			topIdx = i
		}
	}

	layer := BuildErrorLayer(rr, diff, rep, LayerOpts{
		ForceIDs: map[string]bool{target.ID: true},
		MarkID:   target.Index,
	})
	_ = SaveTitledImage(filepath.Join(outDir, "fault_demo.png"),
		"FAULT INJECTION - DAMAGE ATTRIBUTED TO ORIGINAL ID", layer, inkWarn)

	target.Fill = originalFill

	info := &FaultInfo{InjectedID: target.ID, AttributedErrorPixels: topErr}
	if topIdx >= 0 {
		info.AttributedID = scene.All[topIdx].ID
		info.Pass = info.AttributedID == target.ID
	}
	return info
}

func countDrawables(s *Scene) int {
	n := 0
	var walk func(os []*Obj)
	walk = func(os []*Obj) {
		for _, o := range os {
			if len(o.Paths) > 0 {
				n += len(o.Paths)
			}
			walk(o.Children)
		}
	}
	walk(s.Objs)
	return n
}

func buildChecks(scene *Scene, audit *SVGAudit, rep *DiffReport, reloadDelta float64,
	bad *SVGAudit, fault *FaultInfo, browserOK bool) []CheckResult {

	idSet := map[string]bool{}
	for _, id := range audit.IDs {
		idSet[id] = true
	}
	var missing []string
	for _, o := range scene.All {
		if !idSet[o.ID] {
			missing = append(missing, o.ID)
		}
	}
	drawables := countDrawables(scene)

	checks := []CheckResult{
		{
			Name: "all original object IDs survive SVG reload",
			Pass: len(missing) == 0,
			Detail: "expected " + itoa(len(scene.All)) + " ids, reloaded " +
				itoa(audit.IdentifiedElements) + ", missing=" + joinShort(missing),
		},
		{
			Name: "fills / strokes / gradients / clips stay native SVG semantics",
			Pass: audit.GradientElements == len(scene.Grads) &&
				audit.ClipPathElements == len(scene.Clips),
			Detail: "linear+radial gradients=" + itoa(audit.GradientElements) +
				" (want " + itoa(len(scene.Grads)) + "), clipPaths=" +
				itoa(audit.ClipPathElements),
		},
		{
			Name: "reloaded SVG keeps every path separate (not one merged shape)",
			Pass: audit.PathElements == drawables && audit.PathElements >= 10,
			Detail: "path elements=" + itoa(audit.PathElements) +
				", source drawable paths=" + itoa(drawables),
		},
		{
			Name:   "SVG contains no embedded raster image",
			Pass:   audit.ImageElements == 0,
			Detail: "<image> elements=" + itoa(audit.ImageElements),
		},
		{
			Name:   "reload -> raster round-trip stays vector-faithful",
			Pass:   reloadDelta >= 0 && reloadDelta < 2.0,
			Detail: "mean pixel delta own.png vs reloaded.png = " + fnum(reloadDelta),
		},
	}

	if browserOK {
		checks = append(checks, CheckResult{
			Name: "self PNG agrees with browser reference",
			Pass: rep.ErrorFraction < 0.08,
			Detail: "error pixel fraction=" + fnum(rep.ErrorFraction*100) +
				"% threshold=" + fnum(rep.Threshold) +
				", meanDelta=" + fnum(rep.MeanDelta),
		})
	} else {
		checks = append(checks, CheckResult{
			Name:   "browser reference generated",
			Pass:   false,
			Detail: "headless Chrome/Edge capture unavailable",
		})
	}

	if fault != nil {
		checks = append(checks, CheckResult{
			Name: "error layer labels the responsible original object ID",
			Pass: fault.Pass,
			Detail: "injected=" + fault.InjectedID +
				" attributed=" + fault.AttributedID +
				" errorPixels=" + itoa(fault.AttributedErrorPixels),
		})
	}

	if bad != nil {
		reject := bad.ImageElements >= 1 && bad.PathElements == 0
		checks = append(checks, CheckResult{
			Name: "flattened one-bitmap baseline is detected and rejected",
			Pass: reject,
			Detail: "flattened_bad.svg <image>=" + itoa(bad.ImageElements) +
				" path=" + itoa(bad.PathElements) + " (must flag merged bitmap)",
		})
	}
	return checks
}

func joinShort(ss []string) string {
	if len(ss) == 0 {
		return "none"
	}
	out := ss[0]
	for i := 1; i < len(ss) && i < 5; i++ {
		out += "," + ss[i]
	}
	if len(ss) > 5 {
		out += ",..."
	}
	return out
}
