# 20261127

# Normal Decals on Curved Surfaces (Go, zero dependencies)

A self-contained renderer that projects tangent-space normal decals onto a
curved analytic surface and compares correct normal reorientation against the
naive linear average that collapses highlight direction.

## Problem

Averaging the base surface normal and a detail normal in world space:

- pulls every perturbed normal back toward the base normal, so relief
  highlights flatten out (the detail "deflection" is partially erased);
- produces a non-unit vector that must be re-normalized, and drifts further
  when decals overlap.

## What is implemented

- **Normal reorientation (Reoriented Normal Mapping).** Each detail normal is
  stored in tangent space. The shortest-arc quaternion from the +Z axis to the
  current surface normal rotates the detail into world space. The detail's
  highlight direction follows the surface instead of being averaged away
  (`reorient` in `decal.go`).
- **Detail normal blending.** Procedural relief heights (rings, ribs, a bump
  grid) are differentiated by finite differences into tangent-space normals,
  then slerped into the running world normal per decal. Overlapping decals
  chain through the same path and each step ends in an explicit normalization,
  so the result can never leave the unit sphere (`blendDetail`).
- **Edge attenuation.** Coverage uses a Hermite `smoothstep` ramp from full
  detail at 0.75 radius to zero at 1.0 radius (`Decal.project`).
- **Curvature-stable scale.** Every projector is defined in the surface
  tangent frame at its center in **world units**, not screen or UV units.
  Moving a decal from a flat to a sharply curved region keeps an identical
  world-space footprint; an orthographic camera then maps it to a stable
  pixel size (asserted at runtime and in `decal_test.go`).

## Outputs (`out/`)

| File | Content |
| --- | --- |
| `01_original_surface.png` | Undecaled surface with Blinn-Phong lighting |
| `02_decal_normals.png` | Tangent-space detail normals, RGB-encoded, with edge falloff |
| `03_final_lighting_rnm.png` | Final lighting with reoriented decals |
| `04_naive_linear_average.png` | Same scene using a linear normal average, for comparison |
| `05_comparison_split.png` | Left: linear average, right: RNM |

The program also prints:

- projector footprint measurements per decal across low/high curvature areas;
- rendered mask pixel widths with a hard 5% drift assertion;
- a unit-normal audit over every pixel, including a separate maximum for the
  2000+ pixels where two decals overlap.

## Run

```sh
go run ./cmd/decalnormal out   # renders PNGs into ./out
go test ./...                  # unit tests + numeric assertions
```

Only the Go standard library is used (`math`, `image`, `image/png`, `os`);
there are no external modules and no external image assets.
