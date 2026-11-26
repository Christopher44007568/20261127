package main

import (
	"regexp"
	"strconv"
	"strings"
)

var numberRe = regexp.MustCompile(`[-+]?(?:\d+\.?\d*|\.\d+)(?:[eE][-+]?\d+)?`)

// ParsePathD parses the SVG path data subset emitted by PathD
// (M/L/H/V/Q/C/Z, absolute and relative, with implicit repeats).
func ParsePathD(d string) *Path {
	p := &Path{}
	toks := tokenizePath(d)
	pos := 0
	var cur, start Pt

	readNum := func() (float64, bool) {
		if pos >= len(toks) || toks[pos].cmd != 0 {
			return 0, false
		}
		v := toks[pos].num
		pos++
		return v, true
	}

	for pos < len(toks) {
		if toks[pos].cmd == 0 {
			pos++
			continue
		}
		cmd := toks[pos].cmd
		pos++
		rel := cmd >= 'a' && cmd <= 'z'
		up := cmd
		if rel {
			up = cmd - 32
		}

		switch up {
		case 'M', 'L':
			first := true
			for {
				x, ok1 := readNum()
				y, ok2 := readNum()
				if !ok1 || !ok2 {
					break
				}
				pt := Pt{x, y}
				if rel {
					pt = cur.add(pt)
				}
				if up == 'M' && first {
					p.move(pt.X, pt.Y)
					start = pt
					first = false
				} else {
					p.line(pt.X, pt.Y)
				}
				cur = pt
				if up == 'M' {
					up = 'L'
				}
			}
		case 'H', 'V':
			for {
				v, ok := readNum()
				if !ok {
					break
				}
				if up == 'H' {
					x := v
					if rel {
						x = cur.X + v
					}
					cur = Pt{x, cur.Y}
				} else {
					y := v
					if rel {
						y = cur.Y + v
					}
					cur = Pt{cur.X, y}
				}
				p.line(cur.X, cur.Y)
			}
		case 'Q':
			for {
				x1, ok1 := readNum()
				y1, ok2 := readNum()
				x, ok3 := readNum()
				y, ok4 := readNum()
				if !ok1 || !ok2 || !ok3 || !ok4 {
					break
				}
				c1, e := Pt{x1, y1}, Pt{x, y}
				if rel {
					c1 = cur.add(c1)
					e = cur.add(e)
				}
				p.quad(c1.X, c1.Y, e.X, e.Y)
				cur = e
			}
		case 'C':
			for {
				nums := make([]float64, 6)
				ok := true
				for i := range nums {
					nums[i], ok = readNum()
					if !ok {
						break
					}
				}
				if !ok {
					break
				}
				c1 := Pt{nums[0], nums[1]}
				c2 := Pt{nums[2], nums[3]}
				e := Pt{nums[4], nums[5]}
				if rel {
					c1 = cur.add(c1)
					c2 = cur.add(c2)
					e = cur.add(e)
				}
				p.cubic(c1.X, c1.Y, c2.X, c2.Y, e.X, e.Y)
				cur = e
			}
		case 'Z':
			p.close()
			cur = start
		default:
			// Unsupported command: stop rather than mis-parse.
			return p
		}
	}
	return p
}

type pathTok struct {
	cmd byte
	num float64
}

func tokenizePath(d string) []pathTok {
	var toks []pathTok
	i := 0
	for i < len(d) {
		ch := d[i]
		switch {
		case ch == ' ' || ch == ',' || ch == '\t' || ch == '\n' || ch == '\r':
			i++
		case isCmdChar(ch):
			toks = append(toks, pathTok{cmd: ch})
			i++
		default:
			j := i
			if j < len(d) && (d[j] == '+' || d[j] == '-') {
				j++
			}
			dot := false
			for j < len(d) {
				c := d[j]
				if c >= '0' && c <= '9' {
					j++
				} else if c == '.' && !dot {
					dot = true
					j++
				} else if c == 'e' || c == 'E' {
					j++
					if j < len(d) && (d[j] == '+' || d[j] == '-') {
						j++
					}
				} else {
					break
				}
			}
			v, _ := strconv.ParseFloat(strings.TrimSpace(d[i:j]), 64)
			toks = append(toks, pathTok{num: v})
			i = j
		}
	}
	return toks
}

func isCmdChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
