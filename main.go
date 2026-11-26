package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"bipedblend/internal/biped"
	"bipedblend/internal/render"
)

func main() {
	outDir := flag.String("out", "artifacts", "directory for rendered reports")
	flag.Parse()
	if err := run(*outDir); err != nil {
		fmt.Fprintln(os.Stderr, "bipedblend:", err)
		os.Exit(1)
	}
}

func run(outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	result := biped.RunScenario(biped.Scenario{})
	strips := []biped.FilmStrip{
		biped.BuildFilmStrip(0, 12, 4),
		biped.BuildFilmStrip(1.5, 12, 4),
		biped.BuildFilmStrip(4.2, 12, 4),
	}

	framesSVG := render.FilmstripSVG(strips)
	weightsSVG := render.WeightChartSVG(result)
	slipSVG := render.SlipChartSVG(result)
	html := render.HTMLReport(result, framesSVG, weightsSVG, slipSVG)

	files := map[string][]byte{
		"frames.svg":  []byte(framesSVG),
		"weights.svg": []byte(weightsSVG),
		"slip.svg":    []byte(slipSVG),
		"index.html":  []byte(html),
	}
	metrics, err := json.MarshalIndent(map[string]any{
		"summary":        result.Summary,
		"merged_events":  result.Events,
		"triggered_ids":  eventIDs(result.TriggeredEvents),
		"speed_schedule": "0-0.8s:0; 0.8-2.8s:1.5; 2.8-5.2s:4.2; >=5.2s:1.15",
	}, "", "  ")
	if err != nil {
		return err
	}
	files["metrics.json"] = metrics
	timeseries, err := timeseriesCSV(result)
	if err != nil {
		return err
	}
	files["timeseries.csv"] = timeseries

	for name, content := range files {
		path := filepath.Join(outDir, name)
		if err := os.WriteFile(path, content, 0o644); err != nil {
			return err
		}
	}
	fmt.Printf("wrote report to %s/{index.html,frames.svg,weights.svg,slip.svg,metrics.json,timeseries.csv}\n", outDir)
	return nil
}

func timeseriesCSV(result *biped.Result) ([]byte, error) {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	header := []string{
		"frame", "time_s", "target_speed_mps", "smooth_speed_mps", "phase",
		"weight_idle", "weight_walk", "weight_run", "weight_sum",
		"left_raw_slip_mps", "right_raw_slip_mps",
		"left_locked_slip_mps", "right_locked_slip_mps",
		"naive_left_slip_mps", "naive_right_slip_mps",
		"naive_left_cumulative_m", "naive_right_cumulative_m",
	}
	if err := writer.Write(header); err != nil {
		return nil, err
	}
	for i, frame := range result.Frames {
		naive := result.Naive[i]
		record := []string{
			strconv.Itoa(frame.Index),
			strconv.FormatFloat(frame.Time, 'f', 6, 64),
			strconv.FormatFloat(frame.TargetSpeed, 'f', 6, 64),
			strconv.FormatFloat(frame.Speed, 'f', 6, 64),
			strconv.FormatFloat(frame.Phase, 'f', 6, 64),
			strconv.FormatFloat(biped.WeightValue(frame.Weights, "idle"), 'f', 6, 64),
			strconv.FormatFloat(biped.WeightValue(frame.Weights, "walk"), 'f', 6, 64),
			strconv.FormatFloat(biped.WeightValue(frame.Weights, "run"), 'f', 6, 64),
			strconv.FormatFloat(frame.WeightSum, 'f', 12, 64),
			strconv.FormatFloat(frame.LeftSlip.RawSpeed, 'f', 6, 64),
			strconv.FormatFloat(frame.RightSlip.RawSpeed, 'f', 6, 64),
			strconv.FormatFloat(frame.LeftSlip.LockedSpeed, 'e', 6, 64),
			strconv.FormatFloat(frame.RightSlip.LockedSpeed, 'e', 6, 64),
			strconv.FormatFloat(naive.LeftSlipSpeed, 'f', 6, 64),
			strconv.FormatFloat(naive.RightSlipSpeed, 'f', 6, 64),
			strconv.FormatFloat(naive.LeftCumulativeM, 'f', 6, 64),
			strconv.FormatFloat(naive.RightCumulativeM, 'f', 6, 64),
		}
		if err := writer.Write(record); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	return buffer.Bytes(), writer.Error()
}

func eventIDs(events []biped.MergedEvent) []string {
	ids := make([]string, 0, len(events))
	for _, event := range events {
		ids = append(ids, event.ID)
	}
	return ids
}
