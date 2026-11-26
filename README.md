# Skeleton Retarget (Go, standard library only)

Retarget a walking, turning source skeleton onto a target skeleton with
different leg proportions, without foot sliding, hip jumps during the fast
turn, or a screen-pinned centre of pressure. No third-party packages are used.

## What it does

- **Joint mapping** — `JointMap()` in `retarget.go` gives the explicit
  source-to-target joint correspondence; both skeletons share the 19-joint
  hierarchy in `skeleton.go`.
- **Per-bone scale compensation** — the root path is scaled by the leg-length
  ratio (1.02 m / 0.85 m = 1.20), while every target bone keeps its own length
  via the analytic two-bone IK in `ik.go`. Upper-body rotations are transferred
  by re-applying each parent->child direction at the target bone length.
- **Foot locking** — planted soles are pinned to a world-space contact point
  for the whole stance window. Swing ankles are clamped to the reach envelope,
  so the longer legs fold instead of dragging the foot. Contact soles never
  move (asserted in `TestPlantedSolesDoNotSlide`).
- **Turn handling** — heading changes use a smoothstep ramp (finite angular
  velocity) and pelvis motion is low-pass filtered, so the fast 110 degree turn
  produces no hip discontinuity (`TestNoHipJumpDuringTurn`).
- **Centre of pressure** — the CoP is the mean of planted soles and travels in
  both world and screen space (`TestCentreOfPressureMoves`); the damped camera
  never pins it to one screen pixel.

## Layout

- `vec3.go`, `skeleton.go` — math and the two skeletons.
- `ik.go` — analytic two-bone IK, reach envelope, ankle clamping.
- `motion.go`, `motion_build.go` — source gait generation (stride, contacts,
  swing arcs, turn ramp, pelvis recentring and smoothing).
- `retarget.go` — mapping, scale compensation, foot lock and target IK.
- `framebuffer.go`, `font.go`, `renderer.go`, `scene.go`, `chart.go`,
  `palette.go`, `gif.go` — software rendering, bitmap font and GIF encoding
  with a cached nearest-colour LUT (all standard library).
- `metrics.go` — per-frame CSV evidence (soles, contacts, CoP, knee angles,
  pelvis frame-to-frame jump).
- `retarget_test.go` — unit tests for IK, gait, locking, compensation,
  hip continuity and joint-map coverage.

## Run

```sh
go test ./...
go run .
```

Outputs land in `artifacts/`:

- `retarget.gif` — 96-frame side-by-side animation (30 fps).
- `frame_start.png`, `frame_mid_turn.png`, `frame_late.png` — still previews.
- `metrics.csv` — quantitative per-frame evidence.
