package render

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"time"
)

// ShaderVersion changes whenever the shader source or its semantics change.
const ShaderVersion = "shadow-edge-shader-2"

const shaderSource = `
surface(p, mode):
    sky = mix(deep_navy, steel_blue, p.y)
    shadow = ellipse_sdf(p, center=(0.26,-0.42), radii=(0.68,0.23))
    width = mode == hard ? 0.006 : 0.085
    alpha = 1 - smoothstep(-width, width, shadow)
    ground = sky * (1 - 0.72*alpha)
    sphere = circle(p, center=(-0.36,0.08), radius=0.43)
    return composite_sphere(ground, sphere, light=(-0.55,0.82))
`

// Program is a compiled shader variant. The fields below are constants chosen
// at compile time, mirroring a real pipeline's compiled program object.
type Program struct {
	ID            string
	ShaderVersion string
	Mode          ShadowMode
	Penumbra      float64
	CompileMicros int64
}

func CompileShader(mode ShadowMode) Program {
	start := time.Now()

	// Compilation work is deterministic. Timing is recorded for diagnostics
	// only; it is never used to decide whether a cache entry is correct.
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s", ShaderVersion, shaderSource, mode)))
	var penumbra float64
	if mode == ShadowHard {
		penumbra = 0.006
	} else {
		penumbra = 0.085
	}

	// Build the small lookup table a real compiler might emit. The table is
	// deliberately part of the program so the compiler does a little work.
	var lut [256]float64
	for i := range lut {
		x := (float64(i)/255.0)*2 - 1
		if mode == ShadowSoft {
			lut[i] = smoothstep(-1, 1, x)
		} else {
			lut[i] = step(0, x)
		}
	}
	_ = lut

	return Program{
		ID:            hex.EncodeToString(digest[:8]),
		ShaderVersion: ShaderVersion,
		Mode:          mode,
		Penumbra:      penumbra,
		CompileMicros: time.Since(start).Microseconds(),
	}
}

func smoothstep(edge0, edge1, x float64) float64 {
	if edge0 == edge1 {
		if x < edge0 {
			return 0
		}
		return 1
	}
	t := clamp((x-edge0)/(edge1-edge0), 0, 1)
	return t * t * (3 - 2*t)
}

func step(edge, x float64) float64 {
	if x < edge {
		return 0
	}
	return 1
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func mix(a, b, t float64) float64 {
	return a + (b-a)*t
}

func lerpColor(a, b [3]float64, t float64) [3]float64 {
	return [3]float64{
		mix(a[0], b[0], t),
		mix(a[1], b[1], t),
		mix(a[2], b[2], t),
	}
}

func addColor(a, b [3]float64) [3]float64 {
	return [3]float64{a[0] + b[0], a[1] + b[1], a[2] + b[2]}
}

func mulColor(a [3]float64, s float64) [3]float64 {
	return [3]float64{a[0] * s, a[1] * s, a[2] * s}
}

func toByte(v float64) uint8 {
	return uint8(clamp(v, 0, 1)*255 + 0.5)
}

func length2(x, y float64) float64 {
	return math.Hypot(x, y)
}
