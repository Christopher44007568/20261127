package report

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"meshbool/internal/csg"
	"meshbool/internal/mesh"
	"meshbool/internal/render"
	"meshbool/internal/scenes"
)

type OperationView struct {
	Operation    csg.Operation
	Label        string
	English      string
	Symbol       string
	Class        string
	Mesh         mesh.Mesh
	Interior     mesh.Mesh
	SVG          string
	OBJSVG       string
	OBJPath      string
	Volume       float64
	Expected     float64
	AbsError     float64
	Pass         bool
	Triangles    int
	InteriorTris int
	OBJFaces     int
	IOU          float64
	Coverage     float64
}

type CaseView struct {
	scenes.Case
	Operations []OperationView
}

type Summary struct {
	Cases         int
	Operations    int
	Triangles     int
	InteriorFaces int
	MinIOU        float64
	AllPass       bool
	GeneratedAt   string
	TotalOBJFiles int
}

type Page struct {
	Summary Summary
	Cases   []CaseView
}

func Build(outDir string) (Page, error) {
	if err := os.MkdirAll(filepath.Join(outDir, "obj"), 0o755); err != nil {
		return Page{}, err
	}
	page := Page{}
	page.Summary.MinIOU = 1
	page.Summary.AllPass = true

	for _, scene := range scenes.All() {
		view := CaseView{Case: scene}
		for _, operation := range []csg.Operation{csg.Union, csg.Intersection, csg.Difference} {
			result := csg.Apply(scene.A, scene.B, operation)
			camera := render.NewCamera(result.Mesh, 700, 520, 56)
			verifyCamera := render.NewCamera(result.Mesh, 380, 280, 28)
			palette := paletteFor(operation)
			objPath := filepath.Join("obj", fmt.Sprintf("%s-%s.obj", scene.ID, operation))
			if err := mesh.WriteOBJFile(filepath.Join(outDir, objPath), result.Mesh); err != nil {
				return Page{}, err
			}
			roundTrip, err := mesh.ReadOBJFile(filepath.Join(outDir, objPath))
			if err != nil {
				return Page{}, err
			}
			iou := render.Rasterize(verifyCamera, result.Mesh).
				IoU(render.Rasterize(verifyCamera, roundTrip))
			expected := scene.Expected[operation]
			absError := math.Abs(result.Mesh.Volume() - expected)
			pass := absError <= scene.Tolerance && iou >= 0.99999

			opView := OperationView{
				Operation:    operation,
				Label:        operationLabel(operation),
				English:      operationEnglish(operation),
				Symbol:       operationSymbol(operation),
				Class:        string(operation),
				Mesh:         result.Mesh,
				Interior:     result.Interior,
				SVG:          render.SVG(camera, result.Mesh, result.Interior, palette),
				OBJSVG:       render.SVG(camera, roundTrip, mesh.Mesh{}, palette),
				OBJPath:      filepath.ToSlash(objPath),
				Volume:       result.Mesh.Volume(),
				Expected:     expected,
				AbsError:     absError,
				Pass:         pass,
				Triangles:    len(result.Mesh.Triangles),
				InteriorTris: len(result.Interior.Triangles),
				OBJFaces:     len(roundTrip.Triangles),
				IOU:          iou,
				Coverage:     render.Rasterize(verifyCamera, result.Mesh).Coverage(),
			}
			view.Operations = append(view.Operations, opView)
			page.Summary.Triangles += opView.Triangles
			page.Summary.InteriorFaces += opView.InteriorTris
			page.Summary.Operations++
			page.Summary.TotalOBJFiles++
			page.Summary.MinIOU = math.Min(page.Summary.MinIOU, iou)
			page.Summary.AllPass = page.Summary.AllPass && pass
		}
		page.Cases = append(page.Cases, view)
	}
	page.Summary.Cases = len(page.Cases)
	page.Summary.GeneratedAt = time.Now().Format("2006-01-02 15:04 MST")
	if err := os.WriteFile(filepath.Join(outDir, "index.html"), []byte(pageHTML(page)), 0o644); err != nil {
		return Page{}, err
	}
	return page, nil
}

func paletteFor(operation csg.Operation) render.Palette {
	switch operation {
	case csg.Union:
		return render.Palette{Base: "#32d5a4", Edge: "#0a5d4b", Accent: "#59f0c2"}
	case csg.Intersection:
		return render.Palette{Base: "#ffc451", Edge: "#725214", Accent: "#ffd982"}
	default:
		return render.Palette{Base: "#5b9fff", Edge: "#1e4f92", Accent: "#8abaff"}
	}
}

func operationLabel(operation csg.Operation) string {
	switch operation {
	case csg.Union:
		return "并集"
	case csg.Intersection:
		return "交集"
	default:
		return "差集 A − B"
	}
}

func operationEnglish(operation csg.Operation) string {
	switch operation {
	case csg.Union:
		return "UNION"
	case csg.Intersection:
		return "INTERSECTION"
	default:
		return "DIFFERENCE"
	}
}

func operationSymbol(operation csg.Operation) string {
	switch operation {
	case csg.Union:
		return "∪"
	case csg.Intersection:
		return "∩"
	default:
		return "−"
	}
}
