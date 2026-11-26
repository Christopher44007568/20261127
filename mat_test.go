package main

import (
	"math"
	"testing"
)

func TestInverseTranslation(t *testing.T) {
	m := Translate(Vec3{2, -3, 5})
	inv := m.Invert()
	p := inv.TransformPoint(Vec3{2, -3, 5})
	if math.Abs(p.X) > 1e-9 || math.Abs(p.Y) > 1e-9 || math.Abs(p.Z) > 1e-9 {
		t.Fatalf("inverse translation failed: %+v", p)
	}
	prod := m.Mul(inv)
	for i := 0; i < 16; i++ {
		want := 0.0
		if i%5 == 0 {
			want = 1
		}
		if math.Abs(prod.M[i]-want) > 1e-9 {
			t.Fatalf("m * inv != identity at %d: %v", i, prod.M)
		}
	}
}

func TestBindPoseSkinningIdentity(t *testing.T) {
	sk := NewSkeleton()
	mesh := BuildHumanoid(sk)
	world := sk.WorldPose(NewPoseParams(len(sk.Bones)))
	for i, v := range mesh.Vertices {
		got := SkinVertex(v.Pos, v.Weights, world, sk)
		if d := got.Sub(v.Pos).Len(); d > 1e-8 {
			t.Fatalf("vertex %d moved in bind pose: pos=%v got=%v dist=%v",
				i, v.Pos, got, d)
		}
	}
}
