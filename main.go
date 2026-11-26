package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
)

func main() {
	outDir := flag.String("out", "artifacts", "output directory")
	posBits := flag.Uint("pos-bits", 8, "bits per position component (2..31)")
	tolerance := flag.Float64("tolerance", 0.02, "decimation tolerance (metres/chord)")
	maxGap := flag.Int("max-gap", 12, "max frames between retained samples")
	flag.Parse()

	if err := run(*outDir, EncodeOptions{
		PositionBits: uint(*posBits),
		Tolerance:    *tolerance,
		MaxGap:       *maxGap,
	}); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(outDir string, opt EncodeOptions) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	sk := NewSkeleton()
	orig := SynthesizeClip(sk)

	payload, err := Encode(sk, orig, opt)
	if err != nil {
		return err
	}
	decoded, err := Decode(payload)
	if err != nil {
		return err
	}
	comp := decoded.Clip

	badFrame, badJoint := VerifyKeyPoses(sk, orig, comp)
	if badFrame >= 0 {
		return fmt.Errorf("key pose mismatch at frame %d joint %s", badFrame, badJoint)
	}

	metrics, heat := ComputeMetrics(sk, orig, comp)
	slipOrig := ComputeSlip(sk, orig)
	slipComp := ComputeSlip(sk, comp)

	if err := writeFile(filepath.Join(outDir, "original_animation.json"),
		func(f *os.File) error { return WriteClipJSON(f, sk, orig) }); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(outDir, "compressed_animation.json"),
		func(f *os.File) error { return WriteClipJSON(f, sk, comp) }); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "compressed.sakc"), payload, 0o644); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(outDir, "joint_error_heatmap.png"),
		func(f *os.File) error { return RenderHeatmap(f, sk, heat) }); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(outDir, "foot_slip_curve.png"),
		func(f *os.File) error {
			return RenderSlip(f, slipOrig, slipComp, orig.FPS)
		}); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(outDir, "overlay.gif"),
		func(f *os.File) error { return RenderOverlayGIF(f, sk, orig, comp) }); err != nil {
		return err
	}

	stats := buildStats(sk, orig, comp, decoded, payload, opt, metrics, slipComp)
	if err := os.WriteFile(filepath.Join(outDir, "stats.txt"),
		[]byte(stats), 0o644); err != nil {
		return err
	}
	fmt.Print(stats)
	return nil
}

func writeFile(path string, fn func(*os.File) error) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return fn(f)
}

func buildStats(sk *Skeleton, orig, comp *Clip, decoded *Decoded,
	payload []byte, opt EncodeOptions, metrics []JointMetrics, slip SlipCurve) string {
	frames := len(orig.Frames)
	duration := float64(frames) / orig.FPS
	rawSize := frames * JointCount * (3*8 + 4*8)
	bps := float64(len(payload)*8) / duration
	keyCount := 0
	for _, f := range orig.Frames {
		if f.Key {
			keyCount++
		}
	}
	keyBytes := keyCount * JointCount * 7 * 8

	var meanPos, meanAngle, maxPos, maxAngle float64
	for _, m := range metrics {
		meanPos += m.PosMean
		meanAngle += m.AngleMean
		maxPos = math.Max(maxPos, m.PosMax)
		maxAngle = math.Max(maxAngle, m.AngleMaxRad)
	}
	meanPos /= JointCount
	meanAngle = meanAngle / JointCount * 180 / math.Pi

	maxSlip := 0.0
	meanSlipSum, meanSlipN := 0.0, 0
	for _, vals := range [2][]float64{slip.Left, slip.Right} {
		for _, v := range vals {
			if !math.IsNaN(v) {
				maxSlip = math.Max(maxSlip, v)
				meanSlipSum += v
				meanSlipN++
			}
		}
	}
	meanSlip := 0.0
	if meanSlipN > 0 {
		meanSlip = meanSlipSum / float64(meanSlipN)
	}

	s := ""
	s += fmt.Sprintf("frames=%d fps=%.0f duration=%.2fs retained=%d (%.1f%%)\n",
		frames, orig.FPS, duration, len(decoded.Kept),
		100*float64(len(decoded.Kept))/float64(frames))
	s += fmt.Sprintf("position_bits=%d rotation_bits=%d/component tolerance=%.3fm max_gap=%d\n",
		opt.PositionBits, rotBitsPerComp, opt.Tolerance, opt.MaxGap)
	s += fmt.Sprintf("raw_local_bytes=%d compressed_bytes=%d ratio=%.1f:1 bitrate=%.0f bps\n",
		rawSize, len(payload), float64(rawSize)/float64(len(payload)), bps)
	s += fmt.Sprintf("key_frames=%d key_block_bytes=%d (lossless float64, %.0f%% of payload)\n",
		keyCount, keyBytes, 100*float64(keyBytes)/float64(len(payload)))
	s += fmt.Sprintf("mean_joint_pos=%.3fmm max_joint_pos=%.3fmm mean_angle=%.3fdeg max_angle=%.3fdeg\n",
		meanPos*1000, maxPos*1000, meanAngle, maxAngle*180/math.Pi)
	s += fmt.Sprintf("foot_slip_mean=%.3fmm foot_slip_max=%.3fmm\n",
		meanSlip*1000, maxSlip*1000)
	s += "key_pose_roundtrip=exact_float64\n\n"
	s += "per-joint (mean mm / max mm / mean deg / max deg)\n"
	for j, m := range metrics {
		s += fmt.Sprintf("  %-11s %7.3f %7.3f %7.3f %7.3f\n",
			sk.Joints[j].Name, m.PosMean*1000, m.PosMax*1000,
			m.AngleMean*180/math.Pi, m.AngleMaxRad*180/math.Pi)
	}
	return s
}
