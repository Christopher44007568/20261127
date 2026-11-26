package render

import (
	"bytes"
	"testing"

	"meshbool/internal/csg"
	"meshbool/internal/mesh"
	"meshbool/internal/scenes"
)

func TestOBJRoundTripPreservesSilhouette(t *testing.T) {
	for _, scene := range scenes.All() {
		result := csg.Apply(scene.A, scene.B, csg.Union)
		camera := NewCamera(result.Mesh, 320, 240, 24)
		original := Rasterize(camera, result.Mesh)

		var buffer bytes.Buffer
		if err := mesh.WriteOBJ(&buffer, result.Mesh); err != nil {
			t.Fatal(err)
		}
		roundTrip, err := mesh.ReadOBJ(&buffer)
		if err != nil {
			t.Fatal(err)
		}
		reloaded := Rasterize(camera, roundTrip)
		if iou := original.IoU(reloaded); iou < 1 {
			t.Fatalf("%s silhouette IoU = %.12f, want 1", scene.ID, iou)
		}
	}
}

func TestSVGContainsInternalOverlay(t *testing.T) {
	a := mesh.Box(mesh.Vec3{X: -1, Y: -1, Z: -1}, mesh.Vec3{X: 1, Y: 1, Z: 1})
	b := mesh.Box(mesh.Vec3{X: 0, Y: 0, Z: 0}, mesh.Vec3{X: 2, Y: 2, Z: 2})
	result := csg.Apply(a, b, csg.Union)
	camera := NewCamera(result.Mesh, 320, 240, 24)
	svg := SVG(camera, result.Mesh, result.Interior, Palette{Base: "#32d5a4", Edge: "#0a5d4b"})
	if !bytes.Contains([]byte(svg), []byte(`class="interior-layer"`)) {
		t.Fatal("interior overlay layer is missing")
	}
	if !bytes.Contains([]byte(svg), []byte(`clip-path=`)) {
		t.Fatal("interior overlay is not clipped to the result silhouette")
	}
}
