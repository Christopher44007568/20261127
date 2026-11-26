package mesh

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// WriteOBJ exports a welded mesh and preserves numeric precision so a
// round-trip does not change the rendered silhouette.
func WriteOBJ(w io.Writer, m Mesh) error {
	welded, remap := weldMeshForOBJ(m)
	bw := bufio.NewWriter(w)
	if _, err := fmt.Fprintf(bw, "# meshbool triangle mesh\n# vertices %d\n# faces %d\n", len(welded.vertices), len(welded.faces)); err != nil {
		return err
	}
	for _, p := range welded.vertices {
		if _, err := fmt.Fprintf(bw, "v %.17g %.17g %.17g\n", p.X, p.Y, p.Z); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(bw, "g surface"); err != nil {
		return err
	}
	for _, ids := range remap {
		if _, err := fmt.Fprintf(bw, "f %d %d %d\n", ids[0]+1, ids[1]+1, ids[2]+1); err != nil {
			return err
		}
	}
	return bw.Flush()
}

type objData struct {
	vertices []Vec3
	faces    [][3]int
}

func weldMeshForOBJ(m Mesh) (objData, [][3]int) {
	type pointKey struct{ X, Y, Z uint64 }
	index := make(map[pointKey]int)
	data := objData{}
	remap := make([][3]int, 0, len(m.Triangles))
	for _, t := range m.Triangles {
		ids := [3]int{}
		for i, p := range []Vec3{t.A, t.B, t.C} {
			key := pointKey{
				uint64(int64(p.X * 1e12)),
				uint64(int64(p.Y * 1e12)),
				uint64(int64(p.Z * 1e12)),
			}
			id, ok := index[key]
			if !ok {
				id = len(data.vertices)
				index[key] = id
				data.vertices = append(data.vertices, p)
			}
			ids[i] = id
		}
		if ids[0] != ids[1] && ids[1] != ids[2] && ids[2] != ids[0] {
			remap = append(remap, ids)
		}
	}
	data.faces = remap
	return data, remap
}

func WriteOBJFile(path string, m Mesh) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return WriteOBJ(file, m)
}

func ReadOBJ(r io.Reader) (Mesh, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	var vertices []Vec3
	var triangles []Triangle
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		fields := strings.Fields(text)
		switch fields[0] {
		case "v":
			if len(fields) < 4 {
				return Mesh{}, fmt.Errorf("obj line %d: vertex needs 3 coordinates", line)
			}
			x, errX := strconv.ParseFloat(fields[1], 64)
			y, errY := strconv.ParseFloat(fields[2], 64)
			z, errZ := strconv.ParseFloat(fields[3], 64)
			if errX != nil || errY != nil || errZ != nil {
				return Mesh{}, fmt.Errorf("obj line %d: invalid vertex", line)
			}
			vertices = append(vertices, Vec3{x, y, z})
		case "f":
			if len(fields) < 4 {
				return Mesh{}, fmt.Errorf("obj line %d: face needs at least 3 vertices", line)
			}
			ids := make([]int, 0, len(fields)-1)
			for _, field := range fields[1:] {
				parts := strings.Split(field, "/")
				id, err := strconv.Atoi(parts[0])
				if err != nil || id == 0 {
					return Mesh{}, fmt.Errorf("obj line %d: invalid face index %q", line, field)
				}
				if id < 0 {
					id = len(vertices) + id + 1
				}
				if id < 1 || id > len(vertices) {
					return Mesh{}, fmt.Errorf("obj line %d: face index out of range", line)
				}
				ids = append(ids, id-1)
			}
			for i := 1; i+1 < len(ids); i++ {
				t := Triangle{vertices[ids[0]], vertices[ids[i]], vertices[ids[i+1]]}
				if !t.Degenerate() {
					triangles = append(triangles, t)
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return Mesh{}, err
	}
	return Mesh{Triangles: triangles}, nil
}

func ReadOBJFile(path string) (Mesh, error) {
	file, err := os.Open(path)
	if err != nil {
		return Mesh{}, err
	}
	defer file.Close()
	return ReadOBJ(file)
}
