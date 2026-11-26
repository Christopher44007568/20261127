package report

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"featurelines/internal/eval"
	"featurelines/internal/pipeline"
)

type jsonReport struct {
	GeneratedAt string     `json:"generated_at"`
	AllPass     bool       `json:"all_pass"`
	Cases       []jsonCase `json:"cases"`
}

type jsonCase struct {
	ID      string        `json:"id"`
	Title   string        `json:"title"`
	Kind    string        `json:"kind"`
	Pass    bool          `json:"pass"`
	Metrics *eval.Metrics `json:"metrics"`
}

// WriteJSON dumps machine-readable metrics and acceptance results.
func WriteJSON(path string, outs []*pipeline.Output) error {
	rep := jsonReport{GeneratedAt: time.Now().Format(time.RFC3339), AllPass: true}
	for _, o := range outs {
		rep.AllPass = rep.AllPass && o.Metrics.Pass
		rep.Cases = append(rep.Cases, jsonCase{
			ID: o.Case.ID, Title: o.Case.Title, Kind: o.Case.Kind,
			Pass: o.Metrics.Pass, Metrics: o.Metrics,
		})
	}
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dirOf(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// WriteFeatureOBJ writes robust feature chains as OBJ line records.
func WriteFeatureOBJ(path string, outs []*pipeline.Output) error {
	if err := os.MkdirAll(dirOf(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, o := range outs {
		fmt.Fprintf(f, "o %s_features\n", o.Case.ID)
		base := 0
		for _, p := range o.Case.Mesh.V {
			fmt.Fprintf(f, "v %.5f %.5f %.5f\n", p.X, p.Y, p.Z)
		}
		for _, ch := range o.Robust.Creases {
			// OBJ indices are 1-based; emit the ordered chain.
			ids := ""
			for _, vi := range ch.Vertices {
				ids += fmt.Sprintf(" %d", base+vi+1)
			}
			if ch.Closed && len(ch.Vertices) > 0 {
				ids += fmt.Sprintf(" %d", base+ch.Vertices[0]+1)
			}
			fmt.Fprintf(f, "l%s\n", ids)
			for _, gp := range ch.Gaps {
				if len(gp.Points) < 2 {
					continue
				}
				fmt.Fprintf(f, "# bridged gap %.3f mm via %d vertices\n", gp.Length, len(gp.Via))
			}
		}
		base += len(o.Case.Mesh.V)
	}
	return nil
}
