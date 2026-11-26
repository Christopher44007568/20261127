# 20261127

Repository initialized for 20261127.

## tubegen

Generates watertight-ish tubular surfaces from noisy skeleton curves with
per-node radii. Pure Go, standard library only.

### Features

- **Frame propagation**: rotation-minimizing parallel-transport frames along
  each branch, so cross-sections never twist (`frames.go`).
- **Cross-section scaling**: radii are interpolated as a PCHIP (monotone
  cubic) function of arc length — C1 continuous, no overshoot, and invariant
  under resampling (`pchip.go`, `build.go`).
- **No self-intersection**: tube radius is clamped by local centerline
  curvature (smoothed, worst-case over a window) and by clearance to other
  branches, then slope-limited so the radius profile stays smooth
  (`clampRadii` in `build.go`).
- **Joint patching**: each bifurcation gets a filler sphere; child branches
  are trimmed at the sphere and their end ring is projected onto it, so
  branches never overlap and joints stay closed at any branch angle.
- **Rendering** (software rasterizer, `render.go`):
  - `skeleton_overlay.png` — faint tube + resampled centerline (red),
    original noisy nodes (yellow), joint centers (green);
  - `tube_solid.png` — shaded solid;
  - `normal_crack_heatmap.png` — per-vertex normal discontinuity (dihedral
    angles + tube/sphere seam mismatch), jet colormap.
- Also exports `tube.obj` and prints a per-branch safety report
  (min radius-of-curvature / tube-radius, min clearance / tube-radius).

### Usage

```sh
go run .                      # built-in noisy demo skeleton -> out/
go run . -input skel.json -out out -step 0.03 -sides 32
go test ./...                 # continuity / frame / non-overlap tests
```

Input JSON format — a child's first point is the joint and must lie on its
parent's curve:

```json
{"branches": [
  {"id": 0, "parent": -1, "points": [[0,0,0], [0,1,0]], "radii": [0.3, 0.25]},
  {"id": 1, "parent": 0,  "points": [[0,1,0], [0.5,1.8,0]], "radii": [0.15, 0.1]}
]}
```

Flags: `-step` resampling step (0 = auto), `-sides` cross-section sides,
`-smooth` centerline smoothing passes, `-noise`/`-seed` demo generator,
`-az`/`-el` camera, `-size` image size.
