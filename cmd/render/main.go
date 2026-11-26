package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"uid5-0008/internal/pipeline"
)

type manifest struct {
	Frame              string        `json:"frame"`
	Width              int           `json:"width"`
	Height             int           `json:"height"`
	Passes             []passDoc     `json:"passes"`
	DataEdges          []edgeDoc     `json:"data_edges"`
	OrderRequirements  []string      `json:"order_requirements"`
	Composite          string        `json:"composite"`
	AllFramesIdentical bool          `json:"all_frames_identical"`
	Schedules          []scheduleDoc `json:"schedules"`
}

type passDoc struct {
	ID       string   `json:"id"`
	Requires []string `json:"requires"`
	Reads    string   `json:"reads,omitempty"`
	Writes   string   `json:"writes"`
}

type edgeDoc struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Resource string `json:"resource"`
}

type scheduleDoc struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Waves       [][]string `json:"waves"`
	PPM         string     `json:"ppm"`
	SVG         string     `json:"svg"`
	SHA256      string     `json:"sha256"`
}

func main() {
	outDir := flag.String("out", "artifacts", "directory for PPM, SVG, and manifest output")
	flag.Parse()
	if err := run(*outDir); err != nil {
		fmt.Fprintln(os.Stderr, "render:", err)
		os.Exit(1)
	}
}

func run(outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	schedules := pipeline.Schedules()
	reports := make([]scheduleDoc, 0, len(schedules))
	var referenceHash string
	var referenceFrame []byte

	for _, schedule := range schedules {
		outputs, err := pipeline.Execute(schedule)
		if err != nil {
			return err
		}
		frame, err := pipeline.Composite(outputs)
		if err != nil {
			return err
		}

		var ppm bytes.Buffer
		if err := pipeline.WritePPM(&ppm, frame); err != nil {
			return err
		}
		payload := ppm.Bytes()
		sum := sha256.Sum256(payload)
		hash := hex.EncodeToString(sum[:])
		if referenceHash == "" {
			referenceHash = hash
			referenceFrame = append([]byte(nil), payload...)
		} else if hash != referenceHash || !bytes.Equal(referenceFrame, payload) {
			return fmt.Errorf("schedule %q changed the frame (want %s, got %s)", schedule.Name, referenceHash, hash)
		}

		base := "schedule_" + schedule.Name
		ppmName := base + ".ppm"
		svgName := base + ".svg"
		if err := writeBytes(filepath.Join(outDir, ppmName), payload); err != nil {
			return err
		}
		var svg bytes.Buffer
		if err := pipeline.WriteScheduleSVG(&svg, schedule, hash); err != nil {
			return err
		}
		if err := writeBytes(filepath.Join(outDir, svgName), svg.Bytes()); err != nil {
			return err
		}

		reports = append(reports, scheduleDoc{
			Name:        schedule.Name,
			Description: schedule.Description,
			Waves:       waveStrings(schedule.Waves),
			PPM:         ppmName,
			SVG:         svgName,
			SHA256:      hash,
		})
	}

	doc := buildManifest(reports)
	encoded, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if err := writeBytes(filepath.Join(outDir, "schedules.json"), encoded); err != nil {
		return err
	}

	fmt.Printf("wrote %d PPM, %d SVG, and schedules.json to %s\n", len(reports), len(reports), outDir)
	fmt.Printf("all schedules identical: sha256 %s\n", referenceHash)
	return nil
}

func buildManifest(reports []scheduleDoc) manifest {
	passes := make([]passDoc, 0, len(pipeline.AllPasses()))
	for _, id := range pipeline.AllPasses() {
		spec := pipeline.Spec(id)
		requires := make([]string, len(spec.Requires))
		for i, dep := range spec.Requires {
			requires[i] = string(dep)
		}
		passes = append(passes, passDoc{
			ID:       string(id),
			Requires: requires,
			Reads:    spec.Reads,
			Writes:   spec.Writes,
		})
	}

	edges := make([]edgeDoc, 0, len(pipeline.DataEdges()))
	for _, edge := range pipeline.DataEdges() {
		edges = append(edges, edgeDoc{From: string(edge.From), To: string(edge.To), Resource: edge.Resource})
	}

	return manifest{
		Frame:     "single deterministic frame",
		Width:     pipeline.Width,
		Height:    pipeline.Height,
		Passes:    passes,
		DataEdges: edges,
		OrderRequirements: []string{
			"shadow must complete before scene starts",
			"scene must complete before tonemap starts",
			"ui has no data dependency; it may run before, after, or beside tonemap",
			"ui is composited only after every scheduled wave; tonemap never reads UI",
		},
		Composite:          "fixed post-schedule composite of tonemapped scene and straight-alpha UI",
		AllFramesIdentical: reportsIdentical(reports),
		Schedules:          reports,
	}
}

func reportsIdentical(reports []scheduleDoc) bool {
	if len(reports) < 2 {
		return true
	}
	first := reports[0].SHA256
	for _, report := range reports[1:] {
		if report.SHA256 != first {
			return false
		}
	}
	return true
}

func waveStrings(waves []pipeline.Wave) [][]string {
	result := make([][]string, len(waves))
	for i, wave := range waves {
		result[i] = make([]string, len(wave))
		for j, id := range wave {
			result[i][j] = string(id)
		}
	}
	return result
}

func writeBytes(path string, data []byte) error {
	if path == "" {
		return errors.New("empty output path")
	}
	return os.WriteFile(path, data, 0o644)
}
