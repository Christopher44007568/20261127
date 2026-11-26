package main

import (
	"fmt"
	"os"
)

// WriteMetricsCSV dumps per-frame evidence used to audit foot locking,
// weight-point travel and hip continuity across the turn.
func WriteMetricsCSV(path string, src, dst *Skeleton, frames []Frame) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	header := "frame,t,heading,turn_rate," +
		"src_L_contact,src_L_sole_x,src_L_sole_y,src_R_contact,src_R_sole_x,src_R_sole_y," +
		"tgt_L_contact,tgt_L_sole_x,tgt_L_sole_y,tgt_R_contact,tgt_R_sole_x,tgt_R_sole_y," +
		"src_cop_x,src_cop_y,tgt_cop_x,tgt_cop_y," +
		"src_L_knee,src_R_knee,tgt_L_knee,tgt_R_knee,tgt_hip_jump"
	if _, err := fmt.Fprintln(f, header); err != nil {
		return err
	}

	for i, fr := range frames {
		s, t := fr.Source, fr.Target
		hipJump := 0.0
		if i > 0 {
			hipJump = t.Pos[JPelvis].Sub(frames[i-1].Target.Pos[JPelvis]).Len()
		}
		line := fmt.Sprintf(
			"%d,%.3f,%.4f,%.4f,"+
				"%d,%.4f,%.4f,%d,%.4f,%.4f,"+
				"%d,%.4f,%.4f,%d,%.4f,%.4f,"+
				"%.4f,%.4f,%.4f,%.4f,"+
				"%.2f,%.2f,%.2f,%.2f,%.5f",
			i, float64(i)*FrameDT, s.Heading, fr.TurnRate,
			boolInt(s.Contact[0]), s.Sole[0].X, s.Sole[0].Y,
			boolInt(s.Contact[1]), s.Sole[1].X, s.Sole[1].Y,
			boolInt(t.Contact[0]), t.Sole[0].X, t.Sole[0].Y,
			boolInt(t.Contact[1]), t.Sole[1].X, t.Sole[1].Y,
			s.Cop.X, s.Cop.Y, t.Cop.X, t.Cop.Y,
			s.KneeDeg[0], s.KneeDeg[1], t.KneeDeg[0], t.KneeDeg[1], hipJump)
		if _, err := fmt.Fprintln(f, line); err != nil {
			return err
		}
	}
	return nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
