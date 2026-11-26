package render

import "fmt"

// ShadowMode selects the shadow-edge variant produced by the shader.
type ShadowMode uint8

const (
	ShadowHard ShadowMode = iota
	ShadowSoft
)

func (m ShadowMode) String() string {
	switch m {
	case ShadowHard:
		return "hard"
	case ShadowSoft:
		return "soft"
	default:
		return fmt.Sprintf("shadow(%d)", uint8(m))
	}
}

func ParseShadowMode(s string) (ShadowMode, error) {
	switch s {
	case "hard":
		return ShadowHard, nil
	case "soft":
		return ShadowSoft, nil
	default:
		return 0, fmt.Errorf("unknown shadow mode %q", s)
	}
}

// Settings is the complete set of inputs that can change rendered pixels.
// Every field that reaches the shader must also participate in the cache key.
type Settings struct {
	Width        int
	Height       int
	ShadowMode   ShadowMode
	SceneVersion string
}

const (
	DefaultWidth  = 320
	DefaultHeight = 240
	SceneVersion  = "scene-ellipse-v1"
)

func DefaultSettings(mode ShadowMode) Settings {
	return Settings{
		Width:        DefaultWidth,
		Height:       DefaultHeight,
		ShadowMode:   mode,
		SceneVersion: SceneVersion,
	}
}

func (s Settings) Validate() error {
	if s.Width <= 0 || s.Height <= 0 {
		return fmt.Errorf("invalid dimensions %dx%d", s.Width, s.Height)
	}
	if s.ShadowMode != ShadowHard && s.ShadowMode != ShadowSoft {
		return fmt.Errorf("invalid shadow mode %d", s.ShadowMode)
	}
	if s.SceneVersion == "" {
		return fmt.Errorf("scene version is empty")
	}
	return nil
}
