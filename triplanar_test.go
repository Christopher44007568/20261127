package main

import (
	"math"
	"testing"
)

func almostEqual(t *testing.T, name string, got, want, tolerance float64) {
	t.Helper()
	if math.Abs(got-want) > tolerance {
		t.Fatalf("%s = %.12f, want %.12f (tolerance %g)", name, got, want, tolerance)
	}
}

func TestNormalWeightsSumToOne(t *testing.T) {
	normals := []Vec3{
		V(1, 0, 0), V(0, -1, 0), V(0, 0, 1),
		V(1, 1, 1), V(-0.2, 0.9, 0.35), V(0.00001, 0.00002, -1),
	}
	for _, normal := range normals {
		weights := NormalWeights(normal, 4)
		sum := weights.X + weights.Y + weights.Z
		almostEqual(t, "weight sum", sum, 1, 1e-12)
		if weights.X < 0 || weights.Y < 0 || weights.Z < 0 {
			t.Fatalf("negative weight %+v for normal %+v", weights, normal)
		}
	}
}

func TestHardnessSharpensDominantAxis(t *testing.T) {
	normal := V(0.18, 0.91, 0.31)
	soft := NormalWeights(normal, 1)
	hard := NormalWeights(normal, 8)

	if soft.Y <= soft.X || soft.Y <= soft.Z {
		t.Fatalf("dominant Y normal did not receive largest soft weight: %+v", soft)
	}
	if hard.Y <= soft.Y {
		t.Fatalf("hardness did not increase dominant weight: soft=%+v hard=%+v", soft, hard)
	}

	softRatio := soft.Y / soft.X
	hardRatio := hard.Y / hard.X
	if hardRatio <= softRatio {
		t.Fatalf("hardness did not increase axis contrast: soft %.3f hard %.3f", softRatio, hardRatio)
	}
}

func TestWhiteTextureHasNoAdditiveSeamBrightening(t *testing.T) {
	tri := NewTriplanar(ConstantTexture{Color: V(1, 1, 1)}, 0.7, 4)
	for _, normal := range []Vec3{V(1, 1, 1), V(1, 2, 3), V(-1, 1, -2), V(0.45, 0.71, 0.54)} {
		color := tri.Sample(V(12.4, -3.7, 8.2), normal)
		for component, value := range []float64{color.X, color.Y, color.Z} {
			if value > 1+1e-12 || math.Abs(value-1) > 1e-12 {
				t.Fatalf("component %d = %.15f at normal %+v; normalized blend must stay at texture energy", component, value, normal)
			}
		}
	}
}

func TestGravityCoordinateUsesUniformWorldScale(t *testing.T) {
	const scale = 0.83
	p := V(1.7, 2.25, -4.1)
	dy := 0.37

	for _, axis := range []int{AxisX, AxisZ} {
		_, v0 := ProjectionUV(p, axis, scale)
		_, v1 := ProjectionUV(p.Add(V(0, dy, 0)), axis, scale)
		almostEqual(t, "gravity UV delta", v1-v0, dy*scale, 1e-12)
		almostEqual(t, "gravity absolute scale", v1, (p.Y+dy)*scale, 1e-12)
	}

	uX, vX := ProjectionUV(p, AxisX, scale)
	if uX != p.Z*scale || vX != p.Y*scale {
		t.Fatalf("X projection axes changed: got (%g,%g)", uX, vX)
	}
	uZ, vZ := ProjectionUV(p, AxisZ, scale)
	if uZ != p.X*scale || vZ != p.Y*scale {
		t.Fatalf("Z projection axes changed: got (%g,%g)", uZ, vZ)
	}
}

func TestMipmapAndGradientSamplingStayFinite(t *testing.T) {
	mip := NewMipmapTexture(NewRockTexture(32, 9))
	if len(mip.Levels) < 3 {
		t.Fatalf("mip levels = %d, want at least 3", len(mip.Levels))
	}
	for _, lod := range []float64{-1, 0, 0.5, 2, 99} {
		color := mip.SampleLOD(0.37, 0.61, lod)
		if math.IsNaN(color.X+color.Y+color.Z) || color.X < 0 || color.X > 1 {
			t.Fatalf("invalid mip color %+v at LOD %v", color, lod)
		}
	}

	tri := NewTriplanar(mip, 0.6, 5)
	color := tri.SampleGrad(
		V(1, 2, 3),
		V(0.4, 0.8, 0.45),
		V(0.08, 0.01, 0.04),
		V(0.02, 0.09, 0.01),
	)
	sum := color.X + color.Y + color.Z
	if math.IsNaN(sum) || sum < 0 || sum > 3+1e-12 {
		t.Fatalf("invalid gradient triplanar color %+v", color)
	}
}

func TestNegativeScaleAndMirrorNormals(t *testing.T) {
	mesh := Mesh{
		Vertices: []Vertex{{Position: V(1, 2, 3), Normal: V(0.2, 0.8, 0.5)}},
	}
	negative := TransformMesh(mesh, Transform{Scale: V(-1, 1, 1)})
	wantNormal := V(-0.2, 0.8, 0.5).Normalize()
	if negative.Vertices[0].Position.X != -1 {
		t.Fatalf("negative scale position = %+v", negative.Vertices[0].Position)
	}
	if d := negative.Vertices[0].Normal.Sub(wantNormal).Length(); d > 1e-12 {
		t.Fatalf("negative-scale normal = %+v, want %+v", negative.Vertices[0].Normal, wantNormal)
	}

	transformed := TransformMesh(mesh, DefaultTransform())
	mirrored := MirrorMeshX(transformed)
	if mirrored.Vertices[0].Position.X != -transformed.Vertices[0].Position.X {
		t.Fatal("mirror did not reflect position X")
	}
	if mirrored.Vertices[0].Normal.X != -transformed.Vertices[0].Normal.X {
		t.Fatal("mirror did not reflect normal X")
	}
}

func TestRenderFourScenesSmoke(t *testing.T) {
	scenes := BuildScenes(NewRockTexture(64, 7), 0.55, 4)
	if len(scenes) != 4 {
		t.Fatalf("scene count = %d, want 4", len(scenes))
	}
	for _, scene := range scenes {
		frame := renderLinear(scene, 96, 72)
		coverage := 0.0
		for _, depth := range frame.Depth {
			if depth > 0 {
				coverage++
			}
		}
		coverage /= float64(96 * 72)
		if coverage < 0.18 {
			t.Fatalf("%s coverage %.3f is too low", scene.Name, coverage)
		}
	}
}
