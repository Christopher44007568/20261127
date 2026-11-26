package thinfilm

import (
	"math"
	"testing"

	"thinfilm/internal/optconst"
)

func reflectAll(thickness, theta float64) []float64 {
	s := NewStack(1.5, thickness)
	var out []float64
	for lambda := 380.0; lambda <= 780; lambda += 20 {
		out = append(out, s.Reflectance(lambda, theta))
	}
	return out
}

// TestZeroThicknessBareMetal: a 0 nm film must equal the air/metal Fresnel
// reflection at every wavelength and angle (colorless reference).
func TestZeroThicknessBareMetal(t *testing.T) {
	film := NewStack(1.5, 0)
	for _, theta := range []float64{0, 30, 60, 85} {
		for lambda := 380.0; lambda <= 780; lambda += 10 {
			got := film.Reflectance(lambda, theta)
			want := bareMetalReflectance(lambda, theta)
			if math.Abs(got-want) > 1e-10 {
				t.Fatalf("d=0 lambda=%.0f theta=%.0f: got %.6f want %.6f",
					lambda, theta, got, want)
			}
		}
	}
}

// bareMetalReflectance is the direct single-interface Fresnel formula,
// evaluated independently of the characteristic-matrix code path.
func bareMetalReflectance(lambdaNM, thetaDeg float64) float64 {
	theta := clampTheta(thetaDeg)
	n0 := complex(1.0, 0)
	n1 := optconst.FeNK(lambdaNM)
	sin := math.Sin(theta)
	cos0 := complex(math.Cos(theta), 0)
	sin2 := complex(sin*sin, 0)
	cos1 := csqrt(1 - sin2/(n1*n1))

	rs := (n0*cos0 - n1*cos1) / (n0*cos0 + n1*cos1)
	rp := (n1*cos0 - n0*cos1) / (n1*cos0 + n0*cos1)
	return 0.5 * (math.Pow(abs(rs), 2) + math.Pow(abs(rp), 2))
}

func abs(z complex128) float64 { return math.Hypot(real(z), imag(z)) }

// TestGrazingReflection: reflectance stays a valid probability and approaches
// 1 continuously as the angle tends to grazing, for coated and uncoated
// stacks alike. Metals show a pseudo-Brewster minimum at intermediate
// angles, so monotonicity is not required.
func TestGrazingReflection(t *testing.T) {
	for _, d := range []float64{0, 200, 500, 800} {
		s := NewStack(1.5, d)
		for theta := 0.0; theta <= 89.9; theta += 1 {
			r := s.Reflectance(550, theta)
			if r < 0 || r > 1+1e-9 {
				t.Fatalf("invalid R: d=%.0f theta=%.1f r=%.6f",
					d, theta, r)
			}
		}
		if s.Reflectance(550, 89.9) < 0.98 {
			t.Fatalf("grazing R = %.4f, want ~1 (d=%.0f)",
				s.Reflectance(550, 89.9), d)
		}
	}
}

// TestNormalIncidenceContinuity: moving thickness from 200 to 800 nm must
// shift the spectrum smoothly (no jumps) and stay a valid reflectance.
func TestNormalIncidenceContinuity(t *testing.T) {
	s := NewStack(1.5, 200)
	prev := s.Reflectance(550, 0)
	for d := 200.5; d <= 800; d += 0.5 {
		s.ThicknessNM = d
		r := s.Reflectance(550, 0)
		if math.Abs(r-prev) > 0.02 {
			t.Fatalf("spectral jump at d=%.1f: %.4f -> %.4f", d, prev, r)
		}
		if r < 0 || r > 1 {
			t.Fatalf("invalid reflectance %.4f at d=%.1f", r, d)
		}
		prev = r
	}
}

// TestAngularContinuity: reflectance must be continuous in angle.
func TestAngularContinuity(t *testing.T) {
	s := NewStack(1.5, 500)
	prev := s.Reflectance(480, 0)
	for theta := 0.1; theta <= 89.9; theta += 0.1 {
		r := s.Reflectance(480, theta)
		if math.Abs(r-prev) > 0.01 {
			t.Fatalf("angular jump at %.1f deg: %.4f -> %.4f",
				theta, prev, r)
		}
		prev = r
	}
}

// TestFabryPerotOscillation: a thick film shows repeated interference
// extrema; reflectance must not saturate to a constant.
func TestFabryPerotOscillation(t *testing.T) {
	s := NewStack(1.5, 0)
	extrema := 0
	prev := s.Reflectance(550, 0)
	dir := 0.0
	for d := 1.0; d <= 1500; d += 2 {
		s.ThicknessNM = d
		r := s.Reflectance(550, 0)
		delta := r - prev
		if dir > 0 && delta < 0 || dir < 0 && delta > 0 {
			extrema++
		}
		if delta != 0 {
			dir = delta
		}
		prev = r
	}
	// Optical period at 550 nm, n=1.5: lambda/(2n) ~ 183 nm, so 1500 nm
	// spans several fringes.
	if extrema < 6 {
		t.Fatalf("only %d interference extrema, expected several fringes", extrema)
	}
}
