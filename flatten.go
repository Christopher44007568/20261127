package main

import "encoding/base64"

// BuildFlattenedBadSVG produces the anti-pattern baseline: every path merged
// into a single embedded PNG. The reload audit must flag this as a bitmap, not
// as editable vector structure.
func BuildFlattenedBadSVG(png []byte, w, h int) string {
	b64 := base64.StdEncoding.EncodeToString(png)
	return `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="` +
		itoa(w) + `" height="` + itoa(h) + `" viewBox="0 0 ` + itoa(w) + ` ` + itoa(h) + `">
  <title>bad baseline: all vector objects merged into one bitmap</title>
  <image id="merged-bitmap" x="0" y="0" width="` + itoa(w) + `" height="` + itoa(h) + `" ` +
		`href="data:image/png;base64,` + b64 + `"/>
</svg>
`
}
