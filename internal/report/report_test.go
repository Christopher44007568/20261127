package report

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"skinlab/internal/skin"
)

func TestWriteProducesCompleteReport(t *testing.T) {
	mesh := skin.BuildTube()
	bind := skin.Analyze(mesh, skin.ElbowPose("bind", 0, 0))
	maxDistance, exact := skin.BindPoseMismatch(bind.LBS, bind.DQ)
	final := skin.Analyze(mesh, skin.ElbowPose("final", math.Pi/2, 2*math.Pi/3))
	markers := skin.MarkerVertexIndices(mesh)
	trajectories := skin.BuildTrajectories(mesh, markers, math.Pi/2, 2*math.Pi/3, 9)

	outDir := t.TempDir()
	summary, err := Write(outDir, Dataset{
		Mesh:            mesh,
		Bind:            bind,
		Final:           final,
		Trajectory:      trajectories,
		Markers:         markers,
		BindExact:       exact,
		BindMaxDistance: maxDistance,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !summary.BindPoseExact || summary.BindMaxError != 0 {
		t.Fatalf("unexpected bind summary: %+v", summary)
	}
	if summary.LBSMinAreaRatio >= 0.6 || summary.DQMaxAreaRatio <= 1.05 {
		t.Fatalf("report metrics do not show the expected artifacts: %+v", summary)
	}

	expected := []string{"index.html", "area_curve.csv", "trajectories.csv", "deformation.csv", "bind_pose.csv", "summary.json"}
	for _, name := range expected {
		info, err := os.Stat(filepath.Join(outDir, name))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if info.Size() == 0 {
			t.Fatalf("%s is empty", name)
		}
	}

	page, err := os.ReadFile(filepath.Join(outDir, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"LBS", "双四元数", "截面积曲线", "关键顶点轨迹", "展开表面变形热图"} {
		if !strings.Contains(string(page), fragment) {
			t.Fatalf("report is missing %q", fragment)
		}
	}
	if strings.Count(string(page), "<svg") != 4 {
		t.Fatalf("expected four inline SVG figures")
	}

	summaryBytes, err := os.ReadFile(filepath.Join(outDir, "summary.json"))
	if err != nil {
		t.Fatal(err)
	}
	var decoded Summary
	if err := json.Unmarshal(summaryBytes, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.VertexCount != len(mesh.Vertices) || decoded.QuadCount != len(mesh.Quads) {
		t.Fatalf("summary counts do not match mesh: %+v", decoded)
	}
}
