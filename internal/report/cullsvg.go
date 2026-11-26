package report

import (
	"fmt"
	"strings"

	"characterbounds/hierbox"
)

type NamedCull struct {
	Pose   string
	Result hierbox.CullResult
	All    []hierbox.TestedNode
}

func CullingTreeSVG(items []NamedCull) string {
	width, height := 1240, 720
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" font-family="ui-sans-serif,system-ui,'Noto Sans CJK SC','Microsoft YaHei','Droid Sans Fallback',sans-serif">`, width, height, width, height)
	b.WriteString(`<rect width="100%" height="100%" fill="#f8fafc"/>`)
	b.WriteString(`<text x="28" y="38" font-size="22" font-weight="700" fill="#0f172a">Hierarchical frustum culling / 剔除结果</text>`)
	if len(items) > 0 {
		cam := items[0].Result.Camera
		fmt.Fprintf(&b, `<text x="28" y="62" font-size="13" fill="#475569">Camera %s: eye=(%.2f,%.2f,%.2f), target=(%.2f,%.2f,%.2f), fov=%.0f. Red culls, green accepts subtree, amber recurses.</text>`,
			cam.Name, cam.Eye.X, cam.Eye.Y, cam.Eye.Z, cam.Target.X, cam.Target.Y, cam.Target.Z, cam.FOVY)
	}

	colW := 392.0
	for i, item := range items {
		x := 24 + float64(i)*(colW+12)
		drawCullColumn(&b, item, x, 82, colW)
	}
	b.WriteString(`</svg>`)
	return b.String()
}

func drawCullColumn(b *strings.Builder, item NamedCull, x, y, w float64) {
	res := item.Result
	fmt.Fprintf(b, `<rect x="%.1f" y="%.1f" width="%.1f" height="610" rx="12" fill="#ffffff" stroke="#cbd5e1"/>`, x, y, w)
	fmt.Fprintf(b, `<text x="%.1f" y="%.1f" font-size="17" font-weight="800" fill="#0f172a">%s / %s</text>`, x+18, y+28, englishPoseTitle(item.Pose), poseTitle(item.Pose))
	fmt.Fprintf(b, `<text x="%.1f" y="%.1f" font-size="11" fill="#475569">测试 %d/%d 盒；绘制骨盒 %d；剔除 %d；节省 %d 次测试</text>`,
		x+18, y+50, res.Stats.FrustumTests, res.Stats.TotalNodes,
		res.Stats.VisibleBones, res.Stats.CulledBones, res.Stats.SavedFrustumTests)

	tested := make(map[hierbox.BoneID]hierbox.CullStatus)
	for _, n := range res.Tested {
		tested[n.ID] = n.Status
	}
	culled := make(map[hierbox.BoneID]bool)
	for _, id := range res.CulledBones {
		culled[id] = true
	}
	visible := make(map[hierbox.BoneID]bool)
	for _, id := range res.VisibleBones {
		visible[id] = true
	}

	rowY := y + 78
	for _, n := range item.All {
		status := n.Status
		if wasTested, ok := tested[n.ID]; ok {
			status = wasTested
		} else if culled[n.ID] {
			status = "skipped_outside"
		} else if visible[n.ID] {
			status = "skipped_inside"
		}
		drawCullRow(b, x+18, rowY, w-36, n.Depth, n.Name, status, false)
		rowY += 23
	}
	fmt.Fprintf(b, `<text x="%.1f" y="%.1f" font-size="11" fill="#94a3b8">Pale rows were not tested; early traversal saved %d frustum tests.</text>`,
		x+18, y+603, res.Stats.SavedFrustumTests)
}

func drawCullRow(b *strings.Builder, x, y, w float64, depth int, name string, status hierbox.CullStatus, tested bool) {
	indent := float64(depth) * 15
	cx := x + indent
	color := cullColor(status)
	fmt.Fprintf(b, `<rect x="%.1f" y="%.1f" width="%.1f" height="18" rx="4" fill="%s" fill-opacity="0.10"/>`,
		cx-4, y-14, w-indent, color)
	fmt.Fprintf(b, `<circle cx="%.1f" cy="%.1f" r="4.5" fill="%s"/>`, cx+4, y-5, color)
	fmt.Fprintf(b, `<text x="%.1f" y="%.1f" font-size="10.5" fill="#0f172a">%s</text>`, cx+15, y-1, name)
	label := cullLabel(status)
	fmt.Fprintf(b, `<text x="%.1f" y="%.1f" text-anchor="end" font-size="10" font-weight="700" fill="%s">%s</text>`,
		x+w, y-1, color, label)
}

func cullColor(status hierbox.CullStatus) string {
	switch status {
	case hierbox.CullInside:
		return "#16a34a"
	case hierbox.CullIntersecting:
		return "#d97706"
	case hierbox.CullOutside:
		return "#dc2626"
	case "skipped_outside":
		return "#fca5a5"
	case "skipped_inside":
		return "#86efac"
	default:
		return "#64748b"
	}
}

func cullLabel(status hierbox.CullStatus) string {
	switch status {
	case hierbox.CullInside:
		return "inside / accept"
	case hierbox.CullIntersecting:
		return "hit / recurse"
	case hierbox.CullOutside:
		return "outside / cull"
	case "skipped_outside":
		return "with parent / cull"
	case "skipped_inside":
		return "with parent / draw"
	default:
		return string(status)
	}
}
