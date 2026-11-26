package main

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
)

// MetricRow is one foot at one dense simulation sample (long format).
type MetricRow struct {
	Frame            int
	T                float64
	Cycle            float64
	Foot             string
	Stance           bool
	PlantS           float64
	TargetX, TargetY float64
	EndX, EndY       float64
	ErrorM           float64
	LockDriftM       float64
	HipDeg           float64
	KneeDeg          float64
	AnkleDeg         float64
	Status           string
}

// CollectMetrics runs cycles gait cycles with spc samples per cycle.
func CollectMetrics(cfg GaitConfig, cycles, spc int) []MetricRow {
	var rows []MetricRow
	n := cycles * spc
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(spc) * cfg.CycleT * float64(cycles)
		f := cfg.SampleFrame(t)
		for j := 0; j < 2; j++ {
			fs := f.Feet[j]
			nominalHeel := cfg.Slope.Point(fs.PlantS).
				sub(cfg.Slope.Uphill.mul(cfg.HeelBack))
			lockDrift := 0.0
			if fs.Stance {
				lockDrift = fs.Heel.sub(nominalHeel).len()
			}
			rows = append(rows, MetricRow{
				Frame: i, T: t, Cycle: f.Cycle, Foot: fs.Name,
				Stance: fs.Stance, PlantS: fs.PlantS,
				TargetX: fs.Pose.CmdTarget.X, TargetY: fs.Pose.CmdTarget.Y,
				EndX: fs.Pose.Ankle.X, EndY: fs.Pose.Ankle.Y,
				ErrorM: fs.Pose.EndError, LockDriftM: lockDrift,
				HipDeg: deg(fs.Pose.HipFlex), KneeDeg: deg(fs.Pose.KneeFlex),
				AnkleDeg: deg(fs.Pose.AnkleDev), Status: fs.Pose.Status.label(),
			})
		}
	}
	return rows
}

func WriteMetricsCSV(path string, rows []MetricRow) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	head := []string{
		"frame", "t_s", "cycle", "foot", "stance", "plant_s_m",
		"target_x_m", "target_y_m", "end_x_m", "end_y_m",
		"ik_error_m", "sole_lock_drift_m",
		"hip_deg", "knee_deg", "ankle_deg", "status",
	}
	if err := w.Write(head); err != nil {
		return err
	}
	for _, r := range rows {
		rec := []string{
			fmt.Sprintf("%d", r.Frame),
			fmt.Sprintf("%.5f", r.T),
			fmt.Sprintf("%.5f", r.Cycle),
			r.Foot,
			fmt.Sprintf("%t", r.Stance),
			fmt.Sprintf("%.5f", r.PlantS),
			fmt.Sprintf("%.8f", r.TargetX),
			fmt.Sprintf("%.8f", r.TargetY),
			fmt.Sprintf("%.8f", r.EndX),
			fmt.Sprintf("%.8f", r.EndY),
			fmt.Sprintf("%.3e", r.ErrorM),
			fmt.Sprintf("%.3e", r.LockDriftM),
			fmt.Sprintf("%.3f", r.HipDeg),
			fmt.Sprintf("%.3f", r.KneeDeg),
			fmt.Sprintf("%.3f", r.AnkleDeg),
			r.Status,
		}
		if err := w.Write(rec); err != nil {
			return err
		}
	}
	return nil
}

// SweepPoint is one point of a commanded-distance reach sweep.
type SweepPoint struct {
	D       float64
	ErrorM  float64
	KneeDeg float64
	HipDeg  float64
	Status  ReachStatus
}

// ReachSweep commands targets along a fixed forward/down ray across the
// whole distance range, exposing the annulus and the over-reach ramp.
func ReachSweep(cfg GaitConfig) []SweepPoint {
	hip := vec(-0.20, 0.98)
	dir := vec(math.Cos(rad(-55)), math.Sin(rad(-55)))
	var out []SweepPoint
	for i := 0; i <= 105; i++ {
		d := 0.10 + float64(i)*0.01
		target := hip.add(dir.mul(d))
		p := cfg.Leg.Solve(SolveArgs{
			Hip: hip, Target: target, Pole: cfg.Slope.Uphill,
			PelvisAng: cfg.Slope.Ang, FootAng: cfg.Slope.Ang,
		})
		out = append(out, SweepPoint{
			D: d, ErrorM: p.EndError, KneeDeg: deg(p.KneeFlex),
			HipDeg: deg(p.HipFlex), Status: p.Status,
		})
	}
	return out
}

func WriteSweepCSV(path string, pts []SweepPoint) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	if err := w.Write([]string{"cmd_distance_m", "end_error_m", "knee_deg", "hip_deg", "status"}); err != nil {
		return err
	}
	for _, p := range pts {
		if err := w.Write([]string{
			fmt.Sprintf("%.3f", p.D),
			fmt.Sprintf("%.6e", p.ErrorM),
			fmt.Sprintf("%.3f", p.KneeDeg),
			fmt.Sprintf("%.3f", p.HipDeg),
			p.Status.label(),
		}); err != nil {
			return err
		}
	}
	return nil
}

// Summary holds aggregate verification numbers for report.txt.
type Summary struct {
	Samples            int
	MaxIKError         float64
	MaxLockDrift       float64
	KneeMin, KneeMax   float64
	HipMin, HipMax     float64
	AnkleMin, AnkleMax float64
	OverReachFrames    int
	LimitFrames        int
	SweepMaxOverErr    float64
}

func Summarize(cfg GaitConfig, rows []MetricRow, sweep []SweepPoint) Summary {
	s := Summary{KneeMin: 1e9, HipMin: 1e9, AnkleMin: 1e9}
	s.Samples = len(rows)
	for _, r := range rows {
		s.MaxIKError = math.Max(s.MaxIKError, r.ErrorM)
		s.MaxLockDrift = math.Max(s.MaxLockDrift, r.LockDriftM)
		s.KneeMin = math.Min(s.KneeMin, r.KneeDeg)
		s.KneeMax = math.Max(s.KneeMax, r.KneeDeg)
		s.HipMin = math.Min(s.HipMin, r.HipDeg)
		s.HipMax = math.Max(s.HipMax, r.HipDeg)
		s.AnkleMin = math.Min(s.AnkleMin, r.AnkleDeg)
		s.AnkleMax = math.Max(s.AnkleMax, r.AnkleDeg)
		switch r.Status {
		case "OVER-REACH":
			s.OverReachFrames++
		case "FLEX-LIMIT", "JOINT-LIMIT":
			s.LimitFrames++
		}
	}
	for _, p := range sweep {
		if p.Status == StatusOverReach {
			s.SweepMaxOverErr = math.Max(s.SweepMaxOverErr, p.ErrorM)
		}
	}
	return s
}
