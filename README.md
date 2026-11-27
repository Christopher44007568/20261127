# Grazing-angle anisotropic stripe renderer

Pure-Go, zero-dependency renderer for a single textured board viewed at a
grazing angle. The fine diagonal stripes are filtered from a screen-space
derivative footprint instead of being sampled at a fixed 2x2 texel pattern.

## Run

```sh
go run .
go test ./...
```

`go run .` writes every PNG and `metrics.json` into `artifacts/`. The build
uses only the Go standard library.

## Algorithm

1. Intersect the center ray and its four neighboring screen rays with the
   board plane. Central differences of the hit UVs give the Jacobian
   `J = [dUV/dx, dUV/dy]` in texels per screen pixel.
2. The singular vectors of `J * J^T` define the elliptical footprint's major
   and minor axes and its anisotropy ratio.
3. Anisotropic filtering samples along the major axis with Gaussian weights
   (`sigma = major / sqrt(12)`). The minor axis is prefiltered from a mip
   chain at `log2(max(minor, major / probes))`. This keeps the stripe phase
   coherent along the long run while only softening the narrow direction.
4. The preferred probe count is `ceil(2 * major / minor)`. A budget of `B`
   texels permits `floor(B / 4)` bilinear probes; `B = 4` falls back to one
   nearest-mip probe. If the budget cannot cover the preferred count, the
   minor-axis width is raised to `2 * major / probes`, removing
   under-sampled phase instead of folding it into new high-frequency moire.
5. The reference is analytic. The stripe texture is a truncated Fourier
   series, and each harmonic is integrated against the same elliptical
   Gaussian footprint. This provides an alias-free high-quality target for
   the error maps without external image libraries.

The fixed-four baseline always reads the four texels of one bilinear mip-0
fetch. The anisotropic mode defaults to a 128-texel budget.

## Outputs

| File | Contents |
| --- | --- |
| `board_anisotropic.png` | Default 128-texel anisotropic render |
| `board_fixed_four.png` | Fixed-four-texel baseline |
| `board_reference.png` | Analytic elliptical reference |
| `error_heatmap.png` | Anisotropic luma error, x20 |
| `fixed_four_error_heatmap.png` | Fixed-four luma error, x20 |
| `footprint_overlay.png` | 10x screen-space footprint ellipses |
| `tilted_texture.png` | Procedural 1024x1024 fringe texture |
| `budget_degradation.png` | Budgets 4, 8, 16, 32, 64, 128 |
| `comparison.png` | Fixed, anisotropic, reference, error panel |
| `detail_moire.png` | Far-field detail comparison |
| `metrics.json` | RMSE/MAE/PSNR, taps, probes, anisotropy, budgets |

## Current measurements

- Board pixels: `448444`; maximum footprint: `9.67 x 1.29` texels
  (`8.15:1` anisotropy).
- Anisotropic RMSE vs reference: `0.005001`; fixed-four RMSE: `0.014499`.
- Anisotropic long-axis variation is `0.040866`, below the fixed-four
  baseline (`0.042048`), while cross-axis energy drops from `0.017421` to
  `0.016101`.
- The default mode uses at most 17 line probes and `8197624` texel reads.
