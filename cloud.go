package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Cloud struct {
	P    []V3
	N    []V3
	HasN bool
}

func (c *Cloud) Len() int { return len(c.P) }

func ReadCloud(path string) (*Cloud, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".xyz", ".txt", ".asc":
		return ReadXYZ(path)
	case ".ply":
		return ReadPLYFile(path)
	default:
		return nil, fmt.Errorf("不支持的点云格式: %s", filepath.Ext(path))
	}
}

func ReadXYZ(path string) (*Cloud, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	c := &Cloud{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		fields := strings.Fields(line)
		vals := make([]float64, 0, 6)
		for _, field := range fields {
			v, err := strconv.ParseFloat(field, 64)
			if err == nil {
				vals = append(vals, v)
			}
		}
		if len(vals) < 3 {
			continue
		}
		c.P = append(c.P, V3{vals[0], vals[1], vals[2]})
		if len(vals) >= 6 {
			n := Normalize(V3{vals[3], vals[4], vals[5]})
			if Norm(n) > 0.5 {
				c.N = append(c.N, n)
				c.HasN = true
			} else {
				c.N = append(c.N, V3{})
			}
		} else {
			c.N = append(c.N, V3{})
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(c.P) == 0 {
		return nil, fmt.Errorf("未从 %s 读到点", path)
	}
	if !c.HasN {
		c.N = nil
	}
	return c, nil
}

func WriteCloudPLY(path string, c *Cloud) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	fmt.Fprintf(w, "ply\nformat ascii 1.0\nelement vertex %d\n", len(c.P))
	fmt.Fprint(w, "property float x\nproperty float y\nproperty float z\n")
	if c.HasN {
		fmt.Fprint(w, "property float nx\nproperty float ny\nproperty float nz\n")
	}
	fmt.Fprint(w, "end_header\n")
	for i, p := range c.P {
		if c.HasN && i < len(c.N) {
			n := c.N[i]
			fmt.Fprintf(w, "%.7f %.7f %.7f %.6f %.6f %.6f\n", p.X, p.Y, p.Z, n.X, n.Y, n.Z)
		} else {
			fmt.Fprintf(w, "%.7f %.7f %.7f\n", p.X, p.Y, p.Z)
		}
	}
	return w.Flush()
}
