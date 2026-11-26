# 20261127

Repository initialized for 20261127.

## Tile-based software renderer (Go, stdlib only)

A didactic tile-based ("memoryless") renderer modeling how mobile GPUs
handle framebuffer attachments: color, depth and stencil live in tile-local
memory during a pass and only reach the surface through explicit
load/store ops. If a needed store is dropped, the composited frame keeps
stale (black) data along geometry edges — this repo implements the
tile-local decisions that prevent it, plus visualizations.

### Run

```sh
go run .     # writes out/{reference,frame,tiles,discarded,errors}.png
go test ./... # correctness: policy equivalence, MSAA resolve, scissor, stencil
```

### Outputs

- `out/frame.png` — final frame from the tile-local policy (matches reference).
- `out/tiles.png` — tile segmentation of the last pass: green = stored,
  gray = store elided (dead), blue = load elided too.
- `out/discarded.png` — naive full-coverage policy: edge tiles discard
  their store and the frame goes black along silhouettes.
- `out/errors.png` — per-tile diff of the buggy policy vs reference:
  magenta pixels, red borders on corrupted tiles.

### Design

- `tile.go` — load/store op enums, `Tile` (tile-local color/depth/stencil
  sample memory), clip rects.
- `framebuffer.go` — surfaces (color resolved; depth/stencil keep every
  MSAA sample so mid-frame round-trips are lossless) and the tile grid.
- `pass.go` — pass descriptors and the policies: `ReferencePolicy`,
  `TileLocalPolicy` (dead store/load elision; a clear counts as a write;
  partially covered tiles keep their store), `FullCoveragePolicy` (the bug).
- `raster.go` — pass execution (begin/load → draw → resolve/store), 4x
  MSAA triangle rasterizer with depth + stencil tests and scissor window
  clipping. Present always box-filter resolves; raw samples are never
  displayed.
