package main

import (
	"encoding/json"
	"io"
)

type jsonVec3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type jsonQuat struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
	W float64 `json:"w"`
}

type jsonFrame struct {
	Index     int        `json:"index"`
	Time      float64    `json:"time_s"`
	Key       bool       `json:"key"`
	Contacts  [2]bool    `json:"contacts_l_r"`
	Positions []jsonVec3 `json:"positions"`
	Rotations []jsonQuat `json:"rotations"`
}

type jsonClip struct {
	FPS    float64     `json:"fps"`
	Joints []string    `json:"joints"`
	Frames []jsonFrame `json:"frames"`
}

// WriteClipJSON serializes local-space poses for external inspection.
func WriteClipJSON(w io.Writer, sk *Skeleton, clip *Clip) error {
	out := jsonClip{FPS: clip.FPS, Joints: make([]string, JointCount)}
	for j := range sk.Joints {
		out.Joints[j] = sk.Joints[j].Name
	}
	for i, f := range clip.Frames {
		jf := jsonFrame{
			Index:     i,
			Time:      float64(i) / clip.FPS,
			Key:       f.Key,
			Contacts:  f.Contacts,
			Positions: make([]jsonVec3, JointCount),
			Rotations: make([]jsonQuat, JointCount),
		}
		for j := 0; j < JointCount; j++ {
			v, q := f.Pose.Positions[j], f.Pose.Rotations[j]
			jf.Positions[j] = jsonVec3{v.X, v.Y, v.Z}
			jf.Rotations[j] = jsonQuat{q.X, q.Y, q.Z, q.W}
		}
		out.Frames = append(out.Frames, jf)
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
