package qem

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

func LoadOBJ(path string) (*Mesh, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	m, err := ParseOBJ(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	m.Name = strings.TrimSuffix(path, ".obj")
	return m, nil
}

// ParseOBJ reads vertices, faces, and usemtl labels. UV/normals are ignored;
// material labels are retained as seam identifiers.
func ParseOBJ(r io.Reader) (*Mesh, error) {
	m := NewMesh()
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 64*1024*1024)
	material := 0
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		switch fields[0] {
		case "v":
			if len(fields) < 4 {
				return nil, fmt.Errorf("line %d: invalid vertex", lineNo)
			}
			x, err1 := strconv.ParseFloat(fields[1], 64)
			y, err2 := strconv.ParseFloat(fields[2], 64)
			z, err3 := strconv.ParseFloat(fields[3], 64)
			if err1 != nil || err2 != nil || err3 != nil {
				return nil, fmt.Errorf("line %d: invalid vertex coordinates", lineNo)
			}
			m.AddVertex(V(x, y, z))
		case "usemtl":
			if len(fields) >= 2 {
				material = m.AddMaterial(fields[1])
			}
		case "f":
			ids := make([]int, 0, len(fields)-1)
			for _, token := range fields[1:] {
				part := strings.Split(token, "/")[0]
				idx, err := strconv.Atoi(part)
				if err != nil {
					return nil, fmt.Errorf("line %d: invalid face reference %q", lineNo, token)
				}
				if idx < 0 {
					idx = len(m.Vertices) + idx
				} else {
					idx--
				}
				if idx < 0 || idx >= len(m.Vertices) {
					return nil, fmt.Errorf("line %d: vertex index %d out of range", lineNo, idx+1)
				}
				ids = append(ids, idx)
			}
			if len(ids) < 3 {
				return nil, fmt.Errorf("line %d: face needs at least 3 vertices", lineNo)
			}
			for i := 1; i+1 < len(ids); i++ {
				if _, err := m.AddTriangle(ids[0], ids[i], ids[i+1], material); err != nil {
					return nil, fmt.Errorf("line %d: %w", lineNo, err)
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(m.Vertices) == 0 || m.ActiveTriangleCount() == 0 {
		return nil, fmt.Errorf("OBJ contains no triangle geometry")
	}
	return m, nil
}

func (m *Mesh) WriteOBJ(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	bw := bufio.NewWriter(f)
	if _, err := m.WriteOBJTo(bw); err != nil {
		f.Close()
		return err
	}
	if err := bw.Flush(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// WriteOBJTo writes active geometry. Inactive history vertices are omitted and
// active IDs are compacted, which keeps each progressive level a normal OBJ.
func (m *Mesh) WriteOBJTo(w io.Writer) (int, error) {
	idMap := make([]int, len(m.Vertices))
	next := 1
	for i := range m.Vertices {
		if m.Vertices[i].Active {
			idMap[i] = next
			next++
		}
	}
	var written int
	write := func(s string) error {
		n, err := io.WriteString(w, s)
		written += n
		return err
	}
	if err := write("# qemsimplify progressive level\n"); err != nil {
		return written, err
	}
	for _, v := range m.Vertices {
		if !v.Active {
			continue
		}
		if err := write(fmt.Sprintf("v %.8g %.8g %.8g\n", v.Pos.X, v.Pos.Y, v.Pos.Z)); err != nil {
			return written, err
		}
	}
	activeMaterial := -1
	for _, t := range m.Triangles {
		if !t.Active {
			continue
		}
		if t.Material != activeMaterial && t.Material >= 0 && t.Material < len(m.MaterialNames) {
			activeMaterial = t.Material
			if err := write("usemtl " + m.MaterialNames[t.Material] + "\n"); err != nil {
				return written, err
			}
		}
		if err := write(fmt.Sprintf("f %d %d %d\n", idMap[t.A], idMap[t.B], idMap[t.C])); err != nil {
			return written, err
		}
	}
	return written, nil
}
