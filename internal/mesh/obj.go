package mesh

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// ParseOBJ reads vertices and polygonal faces. Polygons are fan-triangulated.
// Vertex texture/normal references and negative indices are supported.
func ParseOBJ(r io.Reader) (*TriangleMesh, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1024), 1<<20)
	out := &TriangleMesh{}
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		switch fields[0] {
		case "v":
			if len(fields) < 4 {
				return nil, fmt.Errorf("line %d: vertex needs three coordinates", lineNo)
			}
			x, err1 := strconv.ParseFloat(fields[1], 64)
			y, err2 := strconv.ParseFloat(fields[2], 64)
			z, err3 := strconv.ParseFloat(fields[3], 64)
			if err1 != nil || err2 != nil || err3 != nil {
				return nil, fmt.Errorf("line %d: invalid vertex coordinate", lineNo)
			}
			out.Vertices = append(out.Vertices, V(x, y, z))
		case "f":
			if len(fields) < 4 {
				continue
			}
			idx := make([]int, 0, len(fields)-1)
			for _, token := range fields[1:] {
				part := token
				if slash := strings.IndexByte(part, '/'); slash >= 0 {
					part = part[:slash]
				}
				v, err := strconv.Atoi(part)
				if err != nil || v == 0 {
					return nil, fmt.Errorf("line %d: invalid face index %q", lineNo, token)
				}
				if v < 0 {
					v = len(out.Vertices) + v + 1
				}
				if v < 1 || v > len(out.Vertices) {
					return nil, fmt.Errorf("line %d: face index %d out of range", lineNo, v)
				}
				idx = append(idx, v-1)
			}
			for i := 1; i+1 < len(idx); i++ {
				out.Faces = append(out.Faces, Tri{idx[0], idx[i], idx[i+1]})
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(out.Faces) == 0 {
		return nil, fmt.Errorf("OBJ contains no faces")
	}
	return out, nil
}

// ReadOBJ loads and welds an OBJ mesh. A relative weld epsilon is used so the
// same file behaves identically at different export scales.
func ReadOBJ(path string) (*TriangleMesh, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	m, err := ParseOBJ(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return m.Weld(1e-7), nil
}

// WriteOBJ writes a polygon mesh. Faces may have different vertex counts.
func WriteOBJ(path string, vertices []Vec3, faces [][]int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	bw := bufio.NewWriter(f)
	fmt.Fprintln(bw, "# meshdiagnostics OBJ")
	for _, p := range vertices {
		fmt.Fprintf(bw, "v %.9g %.9g %.9g\n", p.X, p.Y, p.Z)
	}
	for _, face := range faces {
		fmt.Fprint(bw, "f")
		for _, idx := range face {
			fmt.Fprintf(bw, " %d", idx+1)
		}
		fmt.Fprintln(bw)
	}
	return bw.Flush()
}

// WriteOBJFile writes a triangle mesh.
func (m *TriangleMesh) WriteOBJFile(path string) error {
	faces := make([][]int, len(m.Faces))
	for i, f := range m.Faces {
		faces[i] = []int{f[0], f[1], f[2]}
	}
	return WriteOBJ(path, m.Vertices, faces)
}
