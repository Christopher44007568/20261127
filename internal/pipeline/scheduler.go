package pipeline

import (
	"fmt"
	"sync"
)

// Execute validates and runs every wave. Passes inside one wave are started
// together; passes in later waves wait for the whole predecessor wave.
func Execute(schedule Schedule) (*Outputs, error) {
	if err := schedule.Validate(); err != nil {
		return nil, err
	}

	out := &Outputs{}
	for waveIndex, wave := range schedule.Waves {
		inputs := snapshot(out)
		results := make([]passResult, len(wave))
		errs := make([]error, len(wave))

		var wg sync.WaitGroup
		for i, id := range wave {
			wg.Add(1)
			go func(index int, passID PassID) {
				defer wg.Done()
				results[index], errs[index] = runPass(passID, inputs)
			}(i, id)
		}
		wg.Wait()

		for i, err := range errs {
			if err != nil {
				return nil, fmt.Errorf("%s wave %d pass %s: %w", schedule.Name, waveIndex, wave[i], err)
			}
		}
		merge(out, results)
	}
	return out, nil
}

func snapshot(out *Outputs) passInputs {
	return passInputs{
		shadow:    out.Shadow,
		scene:     out.Scene,
		tonemaped: out.Tonemaped,
		ui:        out.UI,
	}
}

func merge(out *Outputs, results []passResult) {
	for _, result := range results {
		if result.shadow != nil {
			out.Shadow = result.shadow
		}
		if result.scene != nil {
			out.Scene = result.scene
		}
		if result.tonemaped != nil {
			out.Tonemaped = result.tonemaped
		}
		if result.ui != nil {
			out.UI = result.ui
		}
	}
}

func runPass(id PassID, in passInputs) (passResult, error) {
	switch id {
	case Shadow:
		shadow, err := renderShadow()
		return passResult{shadow: shadow}, err
	case Scene:
		if in.shadow == nil {
			return passResult{}, fmt.Errorf("shadow buffer is absent")
		}
		scene, err := renderScene(in.shadow)
		return passResult{scene: scene}, err
	case Tonemap:
		if in.scene == nil {
			return passResult{}, fmt.Errorf("scene HDR buffer is absent")
		}
		return passResult{tonemaped: renderTonemap(in.scene)}, nil
	case UI:
		return passResult{ui: renderUI()}, nil
	default:
		return passResult{}, fmt.Errorf("unknown pass %q", id)
	}
}

// Composite is intentionally not a scheduled pass. It runs once after the DAG
// has resolved, so UI never becomes an input to tonemap.
func Composite(out *Outputs) (Frame, error) {
	if out.Tonemaped == nil {
		return Frame{}, fmt.Errorf("tonemapped scene is absent")
	}
	if out.UI == nil {
		return Frame{}, fmt.Errorf("UI layer is absent")
	}
	if out.Tonemaped.Width != out.UI.Width || out.Tonemaped.Height != out.UI.Height {
		return Frame{}, fmt.Errorf("scene and UI dimensions differ")
	}

	frame := Frame{Width: out.Tonemaped.Width, Height: out.Tonemaped.Height}
	frame.RGB = make([]byte, frame.Width*frame.Height*3)
	for i, scene := range out.Tonemaped.Pixels {
		ui := out.UI.Pixels[i]
		base := [3]float64{scene.X, scene.Y, scene.Z}
		alpha := float64(ui.A) / 255
		overlay := [3]float64{float64(ui.R) / 255, float64(ui.G) / 255, float64(ui.B) / 255}
		for channel := 0; channel < 3; channel++ {
			value := base[channel]*(1-alpha) + overlay[channel]*alpha
			frame.RGB[i*3+channel] = uint8(mathRound(clamp(value, 0, 1) * 255))
		}
	}
	return frame, nil
}

func mathRound(value float64) int {
	return int(value + 0.5)
}
