package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	hf "holefill"
)

func main() {
	outDir := flag.String("out", "outputs", "directory for PPM, OBJ, and report artifacts")
	alpha := flag.Float64("alpha", hf.DefaultPatchOptions().Alpha, "alpha wrapping length scale; zero disables alpha wrapping")
	thickness := flag.Float64("thickness", 0.34, "normal shell thickness for the section PPM")
	noWrap := flag.Bool("no-alpha-wrap", false, "disable optional tangent/alpha wrapping")
	flag.Parse()

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fatal(err)
	}

	original, layout := hf.BuildDemoMesh()
	layout.Thickness = *thickness
	loops, err := hf.ExtractBoundaryLoops(original)
	if err != nil {
		fatal(err)
	}
	hf.ClassifyLoops(original, loops)

	const w, h = 1400, 900
	cam := hf.NewCamera(original, w, h)
	holeCanvas := hf.NewCanvas(w, h, hf.RGB{R: 9, G: 14, B: 27})
	hf.RenderMeshToCanvas(holeCanvas, cam, original, hf.RGB{R: 9, G: 14, B: 27}, "holes")
	hf.DrawAllLoopOverlays(holeCanvas, cam, original, loops, true)
	hf.DrawHoleHUD(holeCanvas, loops)
	mustWritePPM(filepath.Join(*outDir, "01_original_holes.ppm"), holeCanvas)

	opts := hf.DefaultPatchOptions()
	if *noWrap || *alpha <= 0 {
		opts.AlphaWrap = false
	} else {
		opts.AlphaWrap = true
		opts.Alpha = *alpha
	}
	reports, err := hf.PatchAllLoops(original, loops, opts)
	if err != nil {
		fatal(err)
	}

	patchedCanvas := hf.NewCanvas(w, h, hf.RGB{R: 9, G: 14, B: 27})
	hf.RenderMeshToCanvas(patchedCanvas, cam, original, hf.RGB{R: 9, G: 14, B: 27}, "patched")
	hf.DrawPatchHUD(patchedCanvas, reports, opts.AlphaWrap)
	mustWritePPM(filepath.Join(*outDir, "02_patched_mesh.ppm"), patchedCanvas)

	shell := hf.Thicken(original, layout.Thickness)
	section := hf.RenderSectionPPM(shell, layout, 1400, 520)
	mustWritePPM(filepath.Join(*outDir, "03_thickness_section.ppm"), section)

	must(hf.WriteOBJ(filepath.Join(*outDir, "patched_mesh.obj"), original))
	must(hf.WriteOBJ(filepath.Join(*outDir, "thick_shell.obj"), shell))
	mustWriteReport(filepath.Join(*outDir, "report.txt"), loops, reports, opts, original, shell)

	fmt.Printf("wrote artifacts to %s\n", *outDir)
	for _, l := range loops {
		fmt.Println(l.Describe())
	}
	for _, r := range reports {
		fmt.Printf("patched loop=%d class=%s triangles=%d newVertices=%d alphaWrap=%v\n",
			r.LoopID, r.Class, r.Triangles, r.Vertices, r.Wrapped)
	}
	fmt.Printf("top-mesh remaining-outer-boundary-edges=%d (expected); thick-shell-manifold=%v\n",
		len(original.BoundaryEdges()), shell.IsClosedManifold())
}

func mustWritePPM(path string, cv *hf.Canvas) {
	f, err := os.Create(path)
	must(err)
	defer f.Close()
	must(cv.WritePPM(f))
	must(f.Close())
}

func mustWriteReport(path string, loops []*hf.BoundaryLoop, reports []hf.PatchReport, opts hf.PatchOptions, m, shell *hf.Mesh) {
	f, err := os.Create(path)
	must(err)
	defer f.Close()
	fmt.Fprintln(f, "Hole filling report")
	fmt.Fprintf(f, "alpha_wrap=%v alpha=%.3f refine_levels=%d strength=%.3f\n",
		opts.AlphaWrap, opts.Alpha, opts.RefineLevels, opts.Strength)
	fmt.Fprintf(f, "mesh_vertices=%d mesh_triangles=%d remaining_boundary_edges=%d shell_manifold=%v\n\n",
		len(m.V), len(m.T), len(m.BoundaryEdges()), shell.IsClosedManifold())
	for _, l := range loops {
		fmt.Fprintln(f, l.Describe())
	}
	fmt.Fprintln(f)
	for _, r := range reports {
		fmt.Fprintf(f, "patch loop=%d class=%s triangles=%d new_vertices=%d wrapped=%v\n",
			r.LoopID, r.Class, r.Triangles, r.Vertices, r.Wrapped)
	}
	must(f.Close())
}

func must(err error) {
	if err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "holefill:", err)
	os.Exit(1)
}
