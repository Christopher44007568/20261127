package main

import "errors"

// bitWriter packs values MSB-first into a byte stream.
type bitWriter struct {
	buf []byte
	cur byte
	n   uint
}

func (w *bitWriter) WriteBits(v uint64, bits uint) {
	for b := int(bits) - 1; b >= 0; b-- {
		w.cur |= byte((v>>uint(b))&1) << (7 - w.n)
		w.n++
		if w.n == 8 {
			w.buf = append(w.buf, w.cur)
			w.cur, w.n = 0, 0
		}
	}
}

func (w *bitWriter) Bytes() []byte {
	out := append([]byte(nil), w.buf...)
	if w.n > 0 {
		out = append(out, w.cur)
	}
	return out
}

// bitReader is the matching MSB-first reader.
type bitReader struct {
	buf []byte
	pos int
	n   uint
}

var errTruncated = errors.New("compressed stream is truncated")

func (r *bitReader) ReadBits(bits uint) (uint64, error) {
	var v uint64
	for b := uint(0); b < bits; b++ {
		if r.pos >= len(r.buf) {
			return 0, errTruncated
		}
		bit := (r.buf[r.pos] >> (7 - r.n)) & 1
		r.n++
		if r.n == 8 {
			r.pos++
			r.n = 0
		}
		v = (v << 1) | uint64(bit)
	}
	return v, nil
}

// byteBuffer handles the byte-aligned fixed-width parts of the file.
type byteBuffer struct {
	buf []byte
	pos int
}

func (b *byteBuffer) Write(p []byte) { b.buf = append(b.buf, p...) }

func (b *byteBuffer) Read(n int) ([]byte, error) {
	if b.pos+n > len(b.buf) {
		return nil, errTruncated
	}
	out := b.buf[b.pos : b.pos+n]
	b.pos += n
	return out, nil
}
