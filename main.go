package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	outDir := flag.String("out", "artifacts", "directory for PPM files and report")
	flag.Parse()

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fail(err)
	}

	normal := renderScenario("normal", PolicyCorrect, BugNone)
	early := renderScenario("early_release", PolicyEarly, BugEarlyRelease)
	late := renderScenario("delayed_reclaim", PolicyLate, BugLateReclaim)
	reference := renderFreshFrame(1)

	independent := countPixelDiff(normal.Frames[1], reference) == 0
	if !independent {
		fail(fmt.Errorf("normal frame 1 differs from a fresh frame 1"))
	}
	if countPixelDiff(reference, renderIdealFrame(1)) != 0 {
		fail(fmt.Errorf("fresh reference differs from ideal frame 1"))
	}

	outputs := []struct {
		name string
		res  *ScenarioResult
	}{
		{"normal.ppm", normal},
		{"early_release.ppm", early},
		{"delayed_reclaim.ppm", late},
	}
	for _, out := range outputs {
		sheet := buildSheet(out.res, reference, independent)
		path := filepath.Join(*outDir, out.name)
		if err := sheet.WritePPM(path); err != nil {
			fail(fmt.Errorf("write %s: %w", path, err))
		}
	}

	if err := validateDiagnostics(normal, early, late); err != nil {
		fail(err)
	}
	report := buildReport(normal, early, late, independent)
	reportPath := filepath.Join(*outDir, "lifetime_report.txt")
	if err := os.WriteFile(reportPath, []byte(report), 0o644); err != nil {
		fail(err)
	}
	fmt.Print(report)
}

func validateDiagnostics(normal, early, late *ScenarioResult) error {
	if normal.Diffs[0].ChangedPixels != 0 || normal.Diffs[1].ChangedPixels != 0 {
		return fmt.Errorf("normal path has unexpected pixel changes")
	}
	if normal.Stats.FailedAcquires != 0 || normal.Stats.ReusedRetired != 0 {
		return fmt.Errorf("normal path used a failed or stale resource")
	}
	last := early.Diffs[1]
	if early.Stats.FailedAcquires == 0 || last.BlackPixels == 0 || last.MagentaPixels == 0 {
		return fmt.Errorf("early-release diagnostic lacks black/magenta evidence")
	}
	lateLast := late.Diffs[1]
	if late.Stats.ReusedRetired == 0 || lateLast.ChangedPixels == 0 || lateLast.MagentaPixels == 0 {
		return fmt.Errorf("late-reclaim diagnostic lacks stale/magenta evidence")
	}
	if late.QueueEnd == 0 {
		return fmt.Errorf("late-reclaim resource was not still queued")
	}
	return nil
}

func buildReport(normal, early, late *ScenarioResult, independent bool) string {
	var b strings.Builder
	independence := "FAIL"
	if independent {
		independence = "PASS"
	}
	fmt.Fprintf(&b, "renderlifetime report\n")
	fmt.Fprintf(&b, "normal.ppm          diff0=%d diff1=%d independence=%s failed=%d reclaimed=%d\n",
		normal.Diffs[0].ChangedPixels, normal.Diffs[1].ChangedPixels, independence,
		normal.Stats.FailedAcquires, normal.Stats.Reclaimed)
	fmt.Fprintf(&b, "early_release.ppm   diff0=%d diff1=%d black=%d magenta=%d failed=%d\n",
		early.Diffs[0].ChangedPixels, early.Diffs[1].ChangedPixels,
		early.Diffs[1].BlackPixels, early.Diffs[1].MagentaPixels,
		early.Stats.FailedAcquires)
	fmt.Fprintf(&b, "delayed_reclaim.ppm diff0=%d diff1=%d stale=%d magenta=%d reused=%d queue=%d\n",
		late.Diffs[0].ChangedPixels, late.Diffs[1].ChangedPixels,
		late.Diffs[1].ChangedPixels, late.Diffs[1].MagentaPixels,
		late.Stats.ReusedRetired, late.QueueEnd)
	fmt.Fprintf(&b, "policy normal: deferred=%t extra_delay=%d\n",
		normal.Policy.DeferredRelease, normal.Policy.ExtraDelayFrames)
	fmt.Fprintf(&b, "policy early:  deferred=%t extra_delay=%d\n",
		early.Policy.DeferredRelease, early.Policy.ExtraDelayFrames)
	fmt.Fprintf(&b, "policy late:   deferred=%t extra_delay=%d\n",
		late.Policy.DeferredRelease, late.Policy.ExtraDelayFrames)
	return b.String()
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
