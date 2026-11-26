package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
)

// SAKC = skeletal animation keyframe codec, version 1.
//
// Layout (all multi-byte integers little-endian):
//   "SAKC" | u8 version | u8 posBits | u16 frameCount | u16 keptCount
//   f32 fps | skeleton: JointCount*(i8 parent, 3*f32 offset)
//   u32 rangeCount | ranges: 3*f64 min/max per joint
//   kept indices: keptCount*u16
//   u32 keyBlockLen | key block: JointCount*(7 f64) per key, in kept order
//   u32 spanCount | spans: u8 foot, u16 lo, u16 hi each
//   u32 packedLen | bit stream:
//     key flags (keptCount*1), non-key samples
//
// Key samples bypass every quantizer, so key poses decode to identical
// float64 values.

const (
	magicSAKC      = "SAKC"
	codecVersion   = 1
	frameIndexBits = 16
)

// EncodeOptions controls the three independent quantizers.
type EncodeOptions struct {
	PositionBits uint
	Tolerance    float64
	MaxGap       int
}

func putU16(w io.Writer, v int) error {
	var b [2]byte
	binary.LittleEndian.PutUint16(b[:], uint16(v))
	_, err := w.Write(b[:])
	return err
}

func putF32(w io.Writer, v float64) error {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], math.Float32bits(float32(v)))
	_, err := w.Write(b[:])
	return err
}

func putF64(w io.Writer, v float64) error {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], math.Float64bits(v))
	_, err := w.Write(b[:])
	return err
}

// Encode compresses a clip and returns the self-contained binary payload.
func Encode(sk *Skeleton, clip *Clip, opt EncodeOptions) ([]byte, error) {
	if opt.PositionBits < 2 || opt.PositionBits > 31 {
		return nil, errors.New("position bits must be in [2,31]")
	}
	kept := SelectFrames(clip, opt.Tolerance, opt.MaxGap)
	ranges := positionRanges(clip)

	var head bytes.Buffer
	head.WriteString(magicSAKC)
	head.WriteByte(codecVersion)
	head.WriteByte(byte(opt.PositionBits))
	if err := putU16(&head, len(clip.Frames)); err != nil {
		return nil, err
	}
	if err := putU16(&head, len(kept)); err != nil {
		return nil, err
	}
	if err := putF32(&head, clip.FPS); err != nil {
		return nil, err
	}

	for _, j := range sk.Joints {
		head.WriteByte(byte(int8(j.Parent)))
		for _, v := range []float64{j.Offset.X, j.Offset.Y, j.Offset.Z} {
			if err := putF32(&head, v); err != nil {
				return nil, err
			}
		}
	}

	if err := binary.Write(&head, binary.LittleEndian, uint32(JointCount)); err != nil {
		return nil, err
	}
	for j := 0; j < JointCount; j++ {
		for c := 0; c < 3; c++ {
			if err := putF64(&head, ranges[j][c].Min); err != nil {
				return nil, err
			}
			if err := putF64(&head, ranges[j][c].Max); err != nil {
				return nil, err
			}
		}
	}

	for _, idx := range kept {
		if err := putU16(&head, idx); err != nil {
			return nil, err
		}
	}

	var keys bytes.Buffer
	for _, idx := range kept {
		if clip.Frames[idx].Key {
			writeRawPose(&keys, clip.Frames[idx].Pose)
		}
	}

	bw := &bitWriter{}
	for _, idx := range kept {
		bw.WriteBits(boolBit(clip.Frames[idx].Key), 1)
	}

	for _, idx := range kept {
		frame := clip.Frames[idx]
		if frame.Key {
			continue
		}
		for j := 0; j < JointCount; j++ {
			codes := encodePosition(ranges, j, frame.Pose.Positions[j], opt.PositionBits)
			for c := 0; c < 3; c++ {
				bw.WriteBits(codes[c], opt.PositionBits)
			}
		}
		for j := 0; j < JointCount; j++ {
			maxIdx, codes, _ := encodeQuat(frame.Pose.Rotations[j])
			bw.WriteBits(maxIdx, 2)
			for k := 0; k < 3; k++ {
				bw.WriteBits(codes[k], rotBitsPerComp)
			}
		}
	}

	packed := bw.Bytes()
	if err := binary.Write(&head, binary.LittleEndian, uint32(keys.Len())); err != nil {
		return nil, err
	}
	out := append(head.Bytes(), keys.Bytes()...)
	spans := ContactSpans(clip)
	var spanHead bytes.Buffer
	binary.Write(&spanHead, binary.LittleEndian, uint32(len(spans)))
	for _, s := range spans {
		spanHead.WriteByte(byte(s.Foot))
		putU16(&spanHead, s.Lo)
		putU16(&spanHead, s.Hi)
	}
	out = append(out, spanHead.Bytes()...)
	var packedLen [4]byte
	binary.LittleEndian.PutUint32(packedLen[:], uint32(len(packed)))
	out = append(out, packedLen[:]...)
	out = append(out, packed...)
	return out, nil
}

func boolBit(b bool) uint64 {
	if b {
		return 1
	}
	return 0
}

func writeRawPose(w io.Writer, p LocalPose) {
	for j := 0; j < JointCount; j++ {
		putF64(w, p.Positions[j].X)
		putF64(w, p.Positions[j].Y)
		putF64(w, p.Positions[j].Z)
		putF64(w, p.Rotations[j].X)
		putF64(w, p.Rotations[j].Y)
		putF64(w, p.Rotations[j].Z)
		putF64(w, p.Rotations[j].W)
	}
}
