package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

type plyProp struct {
	name   string
	typ    string
	isList bool
	count  string
}

type plyInfo struct {
	format      string
	vertexCount int
	props       []plyProp
}

func ReadPLYFile(path string) (*Cloud, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	marker := []byte("end_header")
	markerAt := bytes.Index(data, marker)
	if markerAt < 0 {
		return nil, fmt.Errorf("%s: 缺少 PLY end_header", path)
	}
	lineEnd := bytes.IndexByte(data[markerAt:], '\n')
	if lineEnd < 0 {
		return nil, fmt.Errorf("%s: PLY 头部不完整", path)
	}
	bodyStart := markerAt + lineEnd + 1
	header := string(data[:markerAt])
	info, err := parsePLYHeader(header)
	if err != nil {
		return nil, err
	}

	c := &Cloud{}
	propIndex := map[string]int{}
	for i, p := range info.props {
		propIndex[p.name] = i
	}

	if info.format == "ascii" {
		sc := bufio.NewScanner(bytes.NewReader(data[bodyStart:]))
		sc.Buffer(make([]byte, 1024*1024), 64*1024*1024)
		for sc.Scan() && len(c.P) < info.vertexCount {
			fields := strings.Fields(sc.Text())
			if len(fields) < 3 {
				continue
			}
			vals := make([]float64, len(info.props))
			for i, p := range info.props {
				if i >= len(fields) {
					break
				}
				v, _ := strconv.ParseFloat(fields[i], 64)
				vals[i] = v
				_ = p
			}
			appendPLYVertex(c, vals, propIndex)
		}
		if err := sc.Err(); err != nil {
			return nil, err
		}
	} else {
		little := info.format == "binary_little_endian"
		off := bodyStart
		for vi := 0; vi < info.vertexCount; vi++ {
			vals := make([]float64, len(info.props))
			for pi, p := range info.props {
				if p.isList {
					n, nsz, err := readPLYScalar(data, off, p.count, little)
					if err != nil {
						return nil, err
					}
					off += nsz
					itemSize := plyTypeSize(p.typ)
					off += int(n) * itemSize
					continue
				}
				v, sz, err := readPLYScalar(data, off, p.typ, little)
				if err != nil {
					return nil, err
				}
				vals[pi] = v
				off += sz
			}
			appendPLYVertex(c, vals, propIndex)
		}
	}

	if len(c.P) == 0 {
		return nil, fmt.Errorf("PLY 中没有顶点: %s", path)
	}
	if !c.HasN {
		c.N = nil
	}
	return c, nil
}

func parsePLYHeader(header string) (plyInfo, error) {
	info := plyInfo{format: "ascii"}
	inVertex := false
	for _, line := range strings.Split(header, "\n") {
		fs := strings.Fields(strings.TrimSpace(line))
		if len(fs) == 0 {
			continue
		}
		switch fs[0] {
		case "format":
			if len(fs) >= 2 {
				info.format = fs[1]
			}
		case "element":
			inVertex = len(fs) >= 3 && fs[1] == "vertex"
			if inVertex {
				n, err := strconv.Atoi(fs[2])
				if err != nil {
					return info, err
				}
				info.vertexCount = n
			}
		case "property":
			if !inVertex || len(fs) < 3 {
				continue
			}
			if fs[1] == "list" && len(fs) >= 5 {
				info.props = append(info.props, plyProp{
					name: fs[4], isList: true, count: fs[2], typ: fs[3],
				})
			} else if fs[1] != "list" {
				info.props = append(info.props, plyProp{name: fs[2], typ: fs[1]})
			}
		}
	}
	return info, nil
}

func appendPLYVertex(c *Cloud, vals []float64, idx map[string]int) {
	get := func(names ...string) (float64, bool) {
		for _, name := range names {
			if i, ok := idx[name]; ok && i < len(vals) {
				return vals[i], true
			}
		}
		return 0, false
	}
	x, okx := get("x")
	y, oky := get("y")
	z, okz := get("z")
	if !okx || !oky || !okz {
		return
	}
	c.P = append(c.P, V3{x, y, z})
	nx, _ := get("nx", "normal_x")
	ny, _ := get("ny", "normal_y")
	nz, _ := get("nz", "normal_z")
	n := V3{nx, ny, nz}
	if Norm(n) > 0.5 {
		c.N = append(c.N, Normalize(n))
		c.HasN = true
	} else {
		c.N = append(c.N, V3{})
	}
}

func plyTypeSize(typ string) int {
	switch typ {
	case "char", "int8", "uchar", "uint8":
		return 1
	case "short", "int16", "ushort", "uint16":
		return 2
	case "int", "int32", "uint", "uint32", "float", "float32":
		return 4
	case "double", "float64":
		return 8
	default:
		return 4
	}
}

func readPLYScalar(b []byte, off int, typ string, little bool) (float64, int, error) {
	sz := plyTypeSize(typ)
	if off+sz > len(b) {
		return 0, 0, fmt.Errorf("PLY 二进制读取越界: type=%s off=%d size=%d body=%d", typ, off, sz, len(b))
	}
	slice := b[off : off+sz]
	u16 := func() uint16 {
		if little {
			return binary.LittleEndian.Uint16(slice)
		}
		return binary.BigEndian.Uint16(slice)
	}
	u32 := func() uint32 {
		if little {
			return binary.LittleEndian.Uint32(slice)
		}
		return binary.BigEndian.Uint32(slice)
	}
	u64 := func() uint64 {
		if little {
			return binary.LittleEndian.Uint64(slice)
		}
		return binary.BigEndian.Uint64(slice)
	}
	switch typ {
	case "char", "int8":
		return float64(int8(slice[0])), 1, nil
	case "uchar", "uint8":
		return float64(slice[0]), 1, nil
	case "short", "int16":
		return float64(int16(u16())), 2, nil
	case "ushort", "uint16":
		return float64(u16()), 2, nil
	case "int", "int32":
		return float64(int32(u32())), 4, nil
	case "uint", "uint32":
		return float64(u32()), 4, nil
	case "float", "float32":
		return float64(math.Float32frombits(u32())), 4, nil
	case "double", "float64":
		return math.Float64frombits(u64()), 8, nil
	default:
		if sz == 4 {
			return float64(int32(u32())), sz, nil
		}
		return 0, sz, fmt.Errorf("不支持的 PLY 类型: %s", typ)
	}
}
