package demo

import (
	"fmt"
	"os"
	"path/filepath"

	"shadowcache/internal/cache"
	"shadowcache/internal/ppm"
	"shadowcache/internal/render"
)

const reportSchema = "shadow-cache-demo/v1"

// Run executes the bug reproduction and the repaired pipeline, writes three
// PPMs plus an audit report, and returns the report.
func Run(outDir string) (*Report, error) {
	if outDir == "" {
		outDir = "artifacts"
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}

	hard := render.DefaultSettings(render.ShadowHard)
	soft := render.DefaultSettings(render.ShadowSoft)

	report := &Report{
		Schema:        reportSchema,
		ShaderVersion: render.ShaderVersion,
		SceneVersion:  render.SceneVersion,
		PolicyNotes: []string{
			"Cache keys carry schema, shader, scene, dimensions, and shadow mode.",
			"Schema/shader/scene changes purge every stale entry before lookup.",
			"A hit is accepted only after its rendered pixels match an uncached reference.",
			"Compile time and lookup time are recorded as diagnostics, never as correctness proof.",
		},
		ValidationNote: "Every cache hit is re-rendered through the no-cache path and compared byte-for-byte.",
	}

	// Ground truth: render the requested soft-shadow frame with no cache.
	correctFrame, _, err := render.RenderDirect(soft)
	if err != nil {
		return nil, err
	}
	correctPath := filepath.Join(outDir, "correct.ppm")
	if err := ppm.WriteBinary(correctPath, correctFrame); err != nil {
		return nil, err
	}

	// Bug reproduction: the legacy key omits shadow mode, so a hard program is
	// returned for the soft request.
	legacy := cache.NewLegacy()
	if _, err := legacy.Render(hard); err != nil {
		return nil, err
	}
	legacySoft, err := legacy.Render(soft)
	if err != nil {
		return nil, err
	}
	wrongPath := filepath.Join(outDir, "wrong_key.ppm")
	if err := ppm.WriteBinary(wrongPath, legacySoft.Frame); err != nil {
		return nil, err
	}
	report.LegacyKey = legacy.KeyFor(soft).String()

	// Fixed pipeline. The hard and soft requests use distinct versioned keys.
	fixed := cache.New(cache.DefaultOptions())
	report.FixedKeyHard = fixed.KeyFor(hard).String()
	report.FixedKeySoft = fixed.KeyFor(soft).String()

	hardMiss, err := fixed.Render(hard)
	if err != nil {
		return nil, err
	}
	softMiss, err := fixed.Render(soft)
	if err != nil {
		return nil, err
	}
	softHit, err := fixed.Render(soft)
	if err != nil {
		return nil, err
	}
	beforeBump := fixed.Stats()
	removed := fixed.BumpSchema()
	rebuilt, err := fixed.Render(soft)
	if err != nil {
		return nil, err
	}
	rebuiltPath := filepath.Join(outDir, "rebuilt.ppm")
	if err := ppm.WriteBinary(rebuiltPath, rebuilt.Frame); err != nil {
		return nil, err
	}
	afterBump := fixed.Stats()

	referenceScenario := Scenario{
		ID:              "correct-reference",
		Description:     "soft-shadow frame rendered with no cache",
		Output:          "correct.ppm",
		RequestedShadow: soft.ShadowMode.String(),
		ReturnedShadow:  render.ShadowSoft.String(),
		Key:             "none|no-cache",
		PixelsCompared:  true,
		ReferenceEqual:  true,
		Diff:            render.Compare(correctFrame, correctFrame),
		PixelHash:       correctFrame.Hash(),
		ReferenceHash:   correctFrame.Hash(),
	}
	report.Scenarios = []Scenario{
		referenceScenario,
		scenarioFromResult("wrong-key-hit", "legacy key omits shadow mode and returns the cached hard shader", "wrong_key.ppm", legacySoft),
		scenarioFromResult("fixed-hard-miss", "fixed key contains shadow hard; first request compiles", "", hardMiss),
		scenarioFromResult("fixed-soft-miss", "fixed key contains shadow soft; mode switch is a miss", "", softMiss),
		scenarioFromResult("fixed-soft-hit", "same soft key hits and its pixels match the uncached reference", "", softHit),
		scenarioFromResult("rebuilt-after-bump", "schema bump purges stale entries; soft request recompiles and verifies", "rebuilt.ppm", rebuilt),
	}

	report.Checks = []Check{
		check("versioned_key_differs_by_shadow_mode", report.FixedKeyHard != report.FixedKeySoft, "hard=%q soft=%q", report.FixedKeyHard, report.FixedKeySoft),
		check("mode_switch_is_a_miss", !softMiss.Hit, "soft request after hard request had cache_hit=%t", softMiss.Hit),
		check("legacy_key_reproduces_wrong_shader", legacySoft.Hit && legacySoft.WrongShader, "legacy hit=%t returned=%s requested=%s", legacySoft.Hit, legacySoft.Program.Mode, soft.ShadowMode),
		check("wrong_key_changes_pixels", !legacySoft.Diff.Equal(), "%s", legacySoft.Diff),
		check("wrong_key_fails_pixel_validation", !legacySoft.Diff.Equal(), "legacy hit pixels match reference=%t", legacySoft.Diff.Equal()),
		check("fixed_hit_is_actually_a_hit", softHit.Hit, "cache_hit=%t", softHit.Hit),
		check("fixed_hit_matches_uncached_reference", softHit.Diff.Equal(), "%s", softHit.Diff),
		check("rebuilt_image_matches_reference", rebuilt.Diff.Equal(), "%s", rebuilt.Diff),
		check("rebuilt_image_matches_correct", render.Compare(rebuilt.Frame, correctFrame).Equal(), "%s", render.Compare(rebuilt.Frame, correctFrame)),
		check("schema_bump_removed_stale_entries", removed >= 2, "removed=%d entries before bump=%d", removed, beforeBump.Entries),
		check("rebuilt_after_invalidation_was_a_miss", !rebuilt.Hit && rebuilt.Rebuilt, "cache_hit=%t rebuilt=%t", rebuilt.Hit, rebuilt.Rebuilt),
		check("hit_records_include_pixel_evidence", hitRecordsHavePixels(softHit), "hit event records compared pixels=%t match=%t", eventBool(softHit, "pixels_compared"), eventBool(softHit, "pixels_match")),
	}
	report.Deterministic = correctFrame.Hash() == softMiss.Frame.Hash() &&
		softMiss.Frame.Hash() == softHit.Frame.Hash() &&
		softHit.Frame.Hash() == rebuilt.Frame.Hash()
	report.AllChecksPass = allChecksPass(report.Checks) && report.Deterministic
	report.FixedCache = CacheReport{Stats: afterBump, Events: fixed.Events()}
	report.LegacyCache = CacheReport{Stats: legacy.Stats(), Events: legacy.Events()}

	report.Outputs.Correct, err = fileInfo(correctPath, correctFrame)
	if err != nil {
		return nil, err
	}
	report.Outputs.WrongKey, err = fileInfo(wrongPath, legacySoft.Frame)
	if err != nil {
		return nil, err
	}
	report.Outputs.Rebuilt, err = fileInfo(rebuiltPath, rebuilt.Frame)
	if err != nil {
		return nil, err
	}

	reportPath := filepath.Join(outDir, "report.json")
	report.Outputs.Report = filepath.ToSlash(reportPath)
	if err := writeJSON(reportPath, report); err != nil {
		return nil, err
	}
	return report, nil
}

func allChecksPass(checks []Check) bool {
	for _, c := range checks {
		if !c.OK {
			return false
		}
	}
	return true
}

func hitRecordsHavePixels(r cache.Result) bool {
	for _, ev := range r.Events {
		if ev.Kind != "hit" {
			continue
		}
		if !ev.PixelsCompared || !ev.PixelsMatch {
			return false
		}
	}
	return len(r.Events) > 0
}

func eventBool(r cache.Result, name string) bool {
	for _, ev := range r.Events {
		if ev.Kind != "hit" {
			continue
		}
		switch name {
		case "pixels_compared":
			return ev.PixelsCompared
		case "pixels_match":
			return ev.PixelsMatch
		}
	}
	return false
}

func Summary(r *Report) string {
	if r == nil {
		return ""
	}
	return fmt.Sprintf("correct=%s wrong_key=%s rebuilt=%s report=%s checks_pass=%t fixed(hits=%d misses=%d invalidations=%d)",
		r.Outputs.Correct.Path, r.Outputs.WrongKey.Path, r.Outputs.Rebuilt.Path, r.Outputs.Report,
		r.AllChecksPass, r.FixedCache.Stats.Hits, r.FixedCache.Stats.Misses, r.FixedCache.Stats.Invalidations)
}
