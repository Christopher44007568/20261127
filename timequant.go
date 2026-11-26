package main

// frameError measures how badly linear/nlerp interpolation between two
// retained samples predicts the intermediate pose at index k. Positions
// contribute metres, rotations contribute a chordal angle proxy.
func frameError(clip *Clip, a, k, b int) float64 {
	u := float64(k-a) / float64(b-a)
	pa := clip.Frames[a].Pose
	pb := clip.Frames[b].Pose
	pk := clip.Frames[k].Pose
	contactK := clip.Frames[k].Contacts
	var worst float64
	for j := 0; j < JointCount; j++ {
		est := lerpVec3(pa.Positions[j], pb.Positions[j], u)
		pe := est.Sub(pk.Positions[j]).Len()
		// Contacts are the most visible low-bitrate artifact, so ankle
		// drift on planted frames gets a much stricter weight.
		if (j == JLAnkle && contactK[0]) || (j == JRAnkle && contactK[1]) {
			pe *= contactWeight
		}
		if pe > worst {
			worst = pe
		}
		estQ := Nlerp(pa.Rotations[j], pb.Rotations[j], u)
		re := 1 - mathAbsDot(estQ, pk.Rotations[j])
		if re > worst {
			worst = re
		}
	}
	return worst
}

const contactWeight = 6.0

func mathAbsDot(a, b Quat) float64 {
	d := a.Dot(b)
	if d < 0 {
		return -d
	}
	return d
}

// SelectFrames performs greedy decimation. Key frames are mandatory anchors:
// they are always retained and interpolation never crosses them, which is
// what lets key poses be restored bit-for-bit.
func SelectFrames(clip *Clip, tol float64, maxGap int) []int {
	n := len(clip.Frames)
	keys := map[int]bool{0: true, n - 1: true}
	for i, f := range clip.Frames {
		if f.Key {
			keys[i] = true
		}
	}
	var bounds []int
	for k := range keys {
		bounds = append(bounds, k)
	}
	sortInts(bounds)

	kept := map[int]bool{0: true}
	for bi := 0; bi+1 < len(bounds); bi++ {
		lo, hi := bounds[bi], bounds[bi+1]
		kept[lo], kept[hi] = true, true
		a := lo
		for a < hi {
			best := hi
			for cand := a + 1; cand <= hi; cand++ {
				if cand-a > maxGap {
					best = cand - 1
					break
				}
				ok := true
				for k := a + 1; k < cand; k++ {
					if frameError(clip, a, k, cand) > tol {
						ok = false
						break
					}
				}
				if ok {
					best = cand
				} else {
					break
				}
			}
			if best == a {
				best = a + 1
			}
			kept[best] = true
			a = best
		}
	}

	var out []int
	for i := 0; i < n; i++ {
		if kept[i] {
			out = append(out, i)
		}
	}
	return out
}

func sortInts(a []int) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j-1] > a[j]; j-- {
			a[j-1], a[j] = a[j], a[j-1]
		}
	}
}
