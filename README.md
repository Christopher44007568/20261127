# Thin-Film Interference Renderer (Go, zero dependencies)

Renders coherent thin-film interference colors on an iron substrate using
only the Go standard library. As the film grows from 0 to 1000 nm the
reflection peaks migrate continuously through the spectrum — there are no
discrete "color jumps".

## What it renders

Running `go run ./cmd/thinfilm` writes three PNGs to `artifacts/`:

- `ramp.png` — thickness (0–1000 nm) vs incidence angle (0° normal to ~89°
  grazing) color field. The zero-thickness column is the colorless bare-metal
  reference and colors stay continuous along both axes.
- `spectrum.png` — reflectance spectra at normal incidence for six film
  thicknesses. Fabry–Pérot fringes shift smoothly with thickness; the bare
  metal curve has no fringes.
- `spheres.png` — shaded bare-iron vs film-coated sphere comparison, with a
  soft contact shadow. Local incidence varies across each surface, so the
  coated sphere shows the angular color shift directly.

## Physical model

- **Stack:** air (`n=1`) / dielectric film (default `n=1.5`, non-magnetic)
  / absorbing iron substrate with wavelength-dependent complex index
  `n + i k`.
- **Solver:** Abeles characteristic-matrix formalism, evaluated separately
  for s and p polarizations; unpolarized reflectance is their mean. Complex
  Snell angles are used throughout, so the absorbing metal is handled
  correctly.
- **Limiting cases:** at `d=0` the layer matrix is the identity and the
  result matches the direct air/metal Fresnel formula exactly (covered by
  `TestZeroThicknessBareMetal`). Angles are clamped just below 90°, keeping
  every `cos(theta)` finite; reflectance converges continuously to ~1 at
  grazing incidence.

## Color pipeline

1. Reflectance is sampled at 5 nm steps over 380–780 nm.
2. Trapezoidal wavelength integration against the CIE 1931 2° standard
   observer and a D65 illuminant gives relative XYZ, normalized by the
   perfect-reflector white point.
3. XYZ → linear sRGB (IEC 61966-2-1 matrix) → sRGB gamma encoding.

The D65 SPD is reconstructed from the normative CIE daylight basis
(`S = S0 + M1*S1 + M2*S2`, CCT 6504 K) instead of shipping a large table.

## Embedded data and regeneration

All runtime data is generated into `internal/optconst/data_gen.go`, so the
renderer itself has zero external dependencies:

- CIE 1931 CMFs, 5 nm grid — JCGT Wyman/Sloan/Shirley supplement (sourced
  from RIT MCSL CIE data).
- Fe `n,k` — Johnson & Christy, *Phys. Rev. B* 9, 5056 (1974), via the
  refractiveindex.info CC0 database.
- CIE daylight basis S0/S1/S2 — colour-science/colour (BSD-3-Clause).

To regenerate, drop the three source files in place (or point at them with
environment variables) and run:

```sh
CMF_FILE=cmf.h FE_FILE=fe.yml DAYLIGHT_FILE=dseries.py \
  go run ./cmd/gendata
```

## Verification

```sh
go test ./...
go vet ./...
```

The suite covers white/black colorimetry, spectral hue ordering, zero-film
Fresnel equivalence, grazing-angle convergence, thickness and angular
continuity, repeated Fabry–Pérot fringes, and font glyph coverage.

## CLI flags

```text
-out string      output directory (default "artifacts")
-film-n float    dielectric refractive index (default 1.5)
-ramp-max float  ramp maximum thickness in nm (default 1000)
-sphere-d float  coated-sphere film thickness in nm (default 500)
```
