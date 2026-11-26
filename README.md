# Skinned Screen-Space Motion Vectors (Go, stdlib only)

A self-contained software rasterizer that generates **per-vertex
post-skinning screen-space motion vectors** for an animated skinned
character, renders the current frame, history reprojection, velocity
map and ghost mask, and contrasts the correct vectors against a
root-only baseline. No third-party packages are used; only the Go
standard library.

## Why

If motion vectors are derived from the root bone only, every vertex
inherits pelvis motion and the arms drag stale history behind them.
Here each vertex is skinned twice per frame — once with the current
bone pose and once with the previous pose — and both are projected.
The screen-space delta is the true vertex velocity:

```
p_cur = Skin(bindPos, weights, worldPose_t)
p_old = Skin(bindPos, weights, worldPose_t-1)
velocity.xy = Project(p_cur).xy - Project(p_old).xy
```

Linear blend skinning uses `p' = sum w_i * W_i * B_i^-1 * p`. The
velocity is interpolated across triangles with perspective-correct
barycentric weights alongside color and depth.

## Run

```
go run .
go test ./...
```

## Outputs (artifacts/)

- `frame_XX.png` — 2x2 contact sheet:
  1. current frame with static HUD,
  2. history reprojection through the velocity buffer,
  3. skinned velocity map (signed R/G, magnitude in B; neutral gray is
     zero, pure black marks HUD pixels that never enter the buffer),
  4. ghost mask (red = ghost/trail edge, yellow = disocclusion,
     blue = valid history correspondence).
- `root_vs_skinned_XX.png` — root-only baseline reprojection next to
  the per-vertex skinned reprojection, with a current-silhouette
  outline and per-region velocity readout.
- `skinning_velocity.gif` — animated contact sheet over the 48-frame
  clip, encoded with `image/gif` and the built-in Plan 9 palette.
- `report.txt` — per-frame speed, ghost/disocclusion counts and the
  hand-region velocity for both methods.

## Animation

- Right arm waves fast (3 cycles per clip) to exercise joint-local
  motion; the torso performs one slow turn. Measured hand-region
  skinned velocity is roughly 9x the root-only value.
- Static HUD panels are drawn through a separate path that writes
  color only and explicitly zeroes velocity, so HUD never enters the
  velocity buffer (asserted in `frame_test.go`).
