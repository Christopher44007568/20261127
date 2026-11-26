package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
)

// Decoded holds the restored clip plus codec metadata for reporting.
type Decoded struct {
	Skeleton     *Skeleton
	Clip         *Clip
	Kept         []int
	PositionBits uint
}

func readU16(r io.Reader) (int, error) {
	var b [2]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0, err
	}
	return int(binary.LittleEndian.Uint16(b[:])), nil
}

func readF32(r io.Reader) (float64, error) {
	var b [4]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0, err
	}
	return float64(math.Float32frombits(binary.LittleEndian.Uint32(b[:]))), nil
}

func readF64(r io.Reader) (float64, error) {
	var b [8]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0, err
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(b[:])), nil
}

// Decode parses an SAKC payload and rebuilds every frame.
func Decode(data []byte) (*Decoded, error) {
	if len(data) < 4 || string(data[:4]) != magicSAKC {
		return nil, errors.New("bad magic")
	}
	r := bytes.NewReader(data)
	r.Seek(4, io.SeekStart)
	version, _ := r.ReadByte()
	if version != codecVersion {
		return nil, errors.New("unsupported codec version")
	}
	posBitsByte, _ := r.ReadByte()
	posBits := uint(posBitsByte)
	frameCount, err := readU16(r)
	if err != nil {
		return nil, err
	}
	keptCount, err := readU16(r)
	if err != nil {
		return nil, err
	}
	fps, err := readF32(r)
	if err != nil {
		return nil, err
	}

	sk := &Skeleton{Joints: make([]Joint, JointCount)}
	for j := 0; j < JointCount; j++ {
		parent, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		off := make([]float64, 3)
		for c := 0; c < 3; c++ {
			off[c], err = readF32(r)
			if err != nil {
				return nil, err
			}
		}
		sk.Joints[j] = Joint{
			Name:   jointName(j),
			Parent: int(int8(parent)),
			Offset: Vec3{off[0], off[1], off[2]},
		}
	}

	var rangeCount uint32
	if err := binary.Read(r, binary.LittleEndian, &rangeCount); err != nil {
		return nil, err
	}
	ranges := make([][3]scalarRange, rangeCount)
	for j := uint32(0); j < rangeCount; j++ {
		for c := 0; c < 3; c++ {
			if ranges[j][c].Min, err = readF64(r); err != nil {
				return nil, err
			}
			if ranges[j][c].Max, err = readF64(r); err != nil {
				return nil, err
			}
		}
	}

	kept := make([]int, keptCount)
	for i := 0; i < keptCount; i++ {
		if kept[i], err = readU16(r); err != nil {
			return nil, err
		}
	}
	var keyBlockLen uint32
	if err := binary.Read(r, binary.LittleEndian, &keyBlockLen); err != nil {
		return nil, err
	}
	keyBlock := make([]byte, keyBlockLen)
	if _, err := io.ReadFull(r, keyBlock); err != nil {
		return nil, err
	}
	keyReader := bytes.NewReader(keyBlock)
	var spanCount uint32
	if err := binary.Read(r, binary.LittleEndian, &spanCount); err != nil {
		return nil, err
	}
	spans := make([]ContactSpan, spanCount)
	for i := uint32(0); i < spanCount; i++ {
		foot, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		lo, err := readU16(r)
		if err != nil {
			return nil, err
		}
		hi, err := readU16(r)
		if err != nil {
			return nil, err
		}
		spans[i] = ContactSpan{Foot: int(foot), Lo: lo, Hi: hi}
	}
	var packedLen uint32
	if err := binary.Read(r, binary.LittleEndian, &packedLen); err != nil {
		return nil, err
	}
	packed := make([]byte, packedLen)
	if _, err := io.ReadFull(r, packed); err != nil {
		return nil, err
	}
	br := &bitReader{buf: packed}

	keyBits := make([]uint64, keptCount)
	for i := 0; i < keptCount; i++ {
		if keyBits[i], err = br.ReadBits(1); err != nil {
			return nil, err
		}
	}
	samples := make([]LocalPose, keptCount)
	for i := 0; i < keptCount; i++ {
		pose := NewLocalPose(sk)
		if keyBits[i] == 1 {
			pose, err = readRawPose(keyReader)
			if err != nil {
				return nil, err
			}
		} else {
			for j := 0; j < JointCount; j++ {
				var codes [3]uint64
				for c := 0; c < 3; c++ {
					if codes[c], err = br.ReadBits(posBits); err != nil {
						return nil, err
					}
				}
				pose.Positions[j] = decodePosition(ranges, j, codes, posBits)
			}
			for j := 0; j < JointCount; j++ {
				maxIdx, err := br.ReadBits(2)
				if err != nil {
					return nil, err
				}
				var codes [3]uint64
				for k := 0; k < 3; k++ {
					if codes[k], err = br.ReadBits(rotBitsPerComp); err != nil {
						return nil, err
					}
				}
				pose.Rotations[j] = decodeQuat(maxIdx, codes)
			}
		}
		samples[i] = pose
	}
	clip := &Clip{FPS: fps, Frames: make([]Frame, frameCount)}
	contacts := ReconstructContacts(frameCount, spans)
	for si := 0; si < keptCount; si++ {
		f := Frame{
			Pose:     samples[si],
			Key:      keyBits[si] == 1,
			Contacts: contacts[kept[si]],
		}
		clip.Frames[kept[si]] = f
	}
	reconstructFrames(clip, kept, samples, fps, contacts)

	return &Decoded{Skeleton: sk, Clip: clip, Kept: kept, PositionBits: posBits}, nil
}

func readRawPose(r io.Reader) (LocalPose, error) {
	p := LocalPose{
		Positions: make([]Vec3, JointCount),
		Rotations: make([]Quat, JointCount),
	}
	for j := 0; j < JointCount; j++ {
		vals := make([]float64, 7)
		for k := 0; k < 7; k++ {
			var err error
			vals[k], err = readF64(r)
			if err != nil {
				return p, err
			}
		}
		p.Positions[j] = Vec3{vals[0], vals[1], vals[2]}
		p.Rotations[j] = Quat{vals[3], vals[4], vals[5], vals[6]}
	}
	return p, nil
}

// reconstructFrames fills non-retained frames by interpolating between the
// surrounding retained samples: lerp for positions, shortest-arc nlerp for
// rotations. Contacts stay in force until the next retained sample.
func reconstructFrames(clip *Clip, kept []int, samples []LocalPose, fps float64, contacts [][2]bool) {
	for si := 0; si+1 < len(kept); si++ {
		a, b := kept[si], kept[si+1]
		for f := a + 1; f < b; f++ {
			u := float64(f-a) / float64(b-a)
			pose := LocalPose{
				Positions: make([]Vec3, JointCount),
				Rotations: make([]Quat, JointCount),
			}
			for j := 0; j < JointCount; j++ {
				pose.Positions[j] = lerpVec3(samples[si].Positions[j], samples[si+1].Positions[j], u)
				pose.Rotations[j] = Nlerp(samples[si].Rotations[j], samples[si+1].Rotations[j], u)
			}
			clip.Frames[f] = Frame{
				Pose:     pose,
				Contacts: contacts[f],
			}
		}
	}
}

func jointName(j int) string {
	names := []string{
		"root", "pelvis", "spine", "chest", "neck", "head",
		"l_shoulder", "l_elbow", "l_hand",
		"r_shoulder", "r_elbow", "r_hand",
		"l_up_leg", "l_leg", "l_ankle",
		"r_up_leg", "r_leg", "r_ankle",
	}
	return names[j]
}
