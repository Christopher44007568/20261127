package render

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"meshbool/internal/mesh"
)

type Palette struct {
	Base   string
	Edge   string
	Accent string
}

type drawTriangle struct {
	P     [3]Projected
	Depth float64
	T     mesh.Triangle
}

// SVG renders a shaded triangle soup with an optional diagnostic interior
// layer. Rendering is a software projection; no JavaScript or GPU is used.
func SVG(camera Camera, surface, interior mesh.Mesh, palette Palette) string {
	var out strings.Builder
	clipID := resultClipID(surface)
	fmt.Fprintf(&out, `<svg class="mesh-svg" viewBox="0 0 %d %d" role="img" aria-label="boolean mesh result">`, camera.Width, camera.Height)
	if len(interior.Triangles) > 0 {
		fmt.Fprintf(&out, `<defs><clipPath id="%s">`, clipID)
		out.WriteString(projectedClip(camera, surface))
		out.WriteString(`</clipPath></defs>`)
	}
	out.WriteString(`<g class="surface-layer">`)
	out.WriteString(renderTriangleLayer(camera, surface, palette, false))
	out.WriteString(`</g>`)
	if len(interior.Triangles) > 0 {
		fmt.Fprintf(&out, `<g class="interior-layer" clip-path="url(#%s)">`, clipID)
		out.WriteString(renderTriangleLayer(camera, interior, palette, true))
		out.WriteString(`</g>`)
	}
	out.WriteString(`</svg>`)
	return out.String()
}

func projectedClip(camera Camera, surface mesh.Mesh) string {
	var out strings.Builder
	for _, triangle := range surface.Triangles {
		a := camera.Project(triangle.A)
		b := camera.Project(triangle.B)
		c := camera.Project(triangle.C)
		fmt.Fprintf(&out, `<polygon points="%.2f,%.2f %.2f,%.2f %.2f,%.2f"/>`,
			a.X, a.Y, b.X, b.Y, c.X, c.Y)
	}
	return out.String()
}

func resultClipID(surface mesh.Mesh) string {
	var hash uint64 = 1469598103934665603
	mix := func(value uint64) {
		hash ^= value
		hash *= 1099511628211
	}
	mix(uint64(len(surface.Triangles)))
	for _, triangle := range surface.Triangles {
		for _, point := range []mesh.Vec3{triangle.A, triangle.B, triangle.C} {
			mix(math.Float64bits(point.X))
			mix(math.Float64bits(point.Y))
			mix(math.Float64bits(point.Z))
		}
	}
	return fmt.Sprintf("mesh-clip-%x", hash)
}

func renderTriangleLayer(camera Camera, m mesh.Mesh, palette Palette, internal bool) string {
	triangles := sortedTriangles(camera, m)
	var out strings.Builder
	for _, triangle := range triangles {
		points := fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f",
			triangle.P[0].X, triangle.P[0].Y,
			triangle.P[1].X, triangle.P[1].Y,
			triangle.P[2].X, triangle.P[2].Y)
		if internal {
			fmt.Fprintf(&out,
				`<polygon points="%s" fill="#ff496d" fill-opacity="0.095" stroke="#ffadbd" stroke-opacity="0.76" stroke-width="0.55"/>`,
				points)
			continue
		}
		shade := faceShade(camera, triangle.T, palette.Base)
		fmt.Fprintf(&out,
			`<polygon points="%s" fill="%s" stroke="%s" stroke-opacity="0.78" stroke-width="0.52" stroke-linejoin="round"/>`,
			points, shade, palette.Edge)
	}
	return out.String()
}

func sortedTriangles(camera Camera, m mesh.Mesh) []drawTriangle {
	out := make([]drawTriangle, 0, len(m.Triangles))
	for _, triangle := range m.Triangles {
		p := [3]Projected{
			camera.Project(triangle.A),
			camera.Project(triangle.B),
			camera.Project(triangle.C),
		}
		out = append(out, drawTriangle{
			P:     p,
			Depth: (p[0].Depth + p[1].Depth + p[2].Depth) / 3,
			T:     triangle,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Depth < out[j].Depth
	})
	return out
}

func faceShade(camera Camera, triangle mesh.Triangle, base string) string {
	normal := triangle.Normal()
	light := mesh.Vec3{X: -0.46, Y: 0.82, Z: 0.64}.Normalize()
	view := camera.Forward
	intensity := 0.58 + 0.32*math.Max(0, normal.Dot(light)) + 0.12*math.Max(0, normal.Dot(view))
	intensity = math.Max(0.42, math.Min(1.08, intensity))
	return mixHex(base, "#f6fbff", (intensity-0.55)*0.42)
}

func mixHex(a, b string, t float64) string {
	t = math.Max(0, math.Min(1, t))
	ar, ag, ab := parseHex(a)
	br, bg, bb := parseHex(b)
	r := int(math.Round(float64(ar)*(1-t) + float64(br)*t))
	g := int(math.Round(float64(ag)*(1-t) + float64(bg)*t))
	bl := int(math.Round(float64(ab)*(1-t) + float64(bb)*t))
	return fmt.Sprintf("#%02x%02x%02x", r, g, bl)
}

func parseHex(value string) (int, int, int) {
	value = strings.TrimPrefix(value, "#")
	if len(value) != 6 {
		return 80, 180, 160
	}
	var r, g, b int
	fmt.Sscanf(value, "%02x%02x%02x", &r, &g, &b)
	return r, g, b
}
