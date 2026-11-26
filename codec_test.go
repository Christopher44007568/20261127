package main

import (
	"math"
	"testing"
)

func testClip(t *testing.T) (*Skeleton, *Clip) {
	t.Helper()
	sk := NewSkeleton()
	return sk, SynthesizeClip(sk)
}

func TestOriginalHasZeroContactSlip(t *testing.T) {
	sk, clip := testClip(t)
	slip := ComputeSlip(sk, clip)
	for foot, vals := range [2][]float64{slip.Left, slip.Right} {
		for f, v := range vals {
			if !math.IsNaN(v) && v > 1e-9 {
				t.Fatalf("foot %d frame %d slips %.2e mm in source clip",
					foot, f, v*1000)
			}
		}
	}
}

func TestKeyPosesRoundTripExactly(t *testing.T) {
	sk, clip := testClip(t)
	for _, bits := range []uint{4, 8, 12} {
		payload, err := Encode(sk, clip, EncodeOptions{
			PositionBits: bits, Tolerance: 0.02, MaxGap: 12,
		})
		if err != nil {
			t.Fatalf("encode bits=%d: %v", bits, err)
		}
		dec, err := Decode(payload)
		if err != nil {
			t.Fatalf("decode bits=%d: %v", bits, err)
		}
		badFrame, badJoint := VerifyKeyPoses(sk, clip, dec.Clip)
		if badFrame >= 0 {
			t.Fatalf("bits=%d key mismatch frame=%d joint=%s",
				bits, badFrame, badJoint)
		}
	}
}

func TestContactsRoundTrip(t *testing.T) {
	sk, clip := testClip(t)
	payload, err := Encode(sk, clip, EncodeOptions{
		PositionBits: 8, Tolerance: 0.02, MaxGap: 12,
	})
	if err != nil {
		t.Fatal(err)
	}
	dec, err := Decode(payload)
	if err != nil {
		t.Fatal(err)
	}
	for f := range clip.Frames {
		if clip.Frames[f].Contacts != dec.Clip.Frames[f].Contacts {
			t.Fatalf("contact mismatch at frame %d: orig=%v comp=%v",
				f, clip.Frames[f].Contacts, dec.Clip.Frames[f].Contacts)
		}
	}
}

func TestFrameCountPreserved(t *testing.T) {
	sk, clip := testClip(t)
	payload, err := Encode(sk, clip, EncodeOptions{
		PositionBits: 8, Tolerance: 0.02, MaxGap: 12,
	})
	if err != nil {
		t.Fatal(err)
	}
	dec, err := Decode(payload)
	if err != nil {
		t.Fatal(err)
	}
	if len(dec.Clip.Frames) != len(clip.Frames) {
		t.Fatalf("frame count %d != %d",
			len(dec.Clip.Frames), len(clip.Frames))
	}
}

func TestBitIOWRoundTrip(t *testing.T) {
	bw := &bitWriter{}
	values := []struct {
		v    uint64
		bits uint
	}{
		{5, 3}, {0, 1}, {12345, 14}, {7, 3}, {1, 1}, {4095, 12},
	}
	for _, x := range values {
		bw.WriteBits(x.v, x.bits)
	}
	br := &bitReader{buf: bw.Bytes()}
	for i, x := range values {
		got, err := br.ReadBits(x.bits)
		if err != nil {
			t.Fatalf("read %d: %v", i, err)
		}
		if got != x.v {
			t.Fatalf("value %d: got %d want %d", i, got, x.v)
		}
	}
}

func TestQuatRoundTrip(t *testing.T) {
	cases := []Quat{
		{W: 1},
		RotX(0.7),
		RotY(-1.3).Mul(RotX(0.4)),
		{X: 0.1, Y: -0.2, Z: 0.3, W: 0.927},
	}
	for _, q := range cases {
		maxIdx, codes, _ := encodeQuat(q)
		got := decodeQuat(maxIdx, codes)
		if angle := quatAngle(q.Normalized(), got); angle > 0.02 {
			t.Fatalf("quat round trip angle %.4f rad too large", angle)
		}
	}
}

func TestRetainedFramesRespectMaxGap(t *testing.T) {
	_, clip := testClip(t)
	kept := SelectFrames(clip, 0.02, 12)
	for i := 1; i < len(kept); i++ {
		if kept[i]-kept[i-1] > 12 {
			t.Fatalf("gap %d exceeds max at index %d",
				kept[i]-kept[i-1], i)
		}
	}
}
