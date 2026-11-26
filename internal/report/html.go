package report

import (
	"fmt"
	"html"
	"strings"
)

func pageHTML(page Page) string {
	var out strings.Builder
	out.WriteString(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="color-scheme" content="dark">
<title>Tri-Mesh Boolean Lab · Pure Go</title>
<style>`)
	out.WriteString(pageCSS())
	out.WriteString(`</style>
</head>
<body>
<div class="shell">
`)
	writeHeader(&out, page)
	writeLegend(&out)
	for _, view := range page.Cases {
		writeCase(&out, view)
	}
	writeMethod(&out)
	writeVerification(&out, page)
	out.WriteString(`<footer class="footer">
  <span><b>TRI-MESH BOOLEAN LAB</b> · BSP / GO STANDARD LIBRARY</span>
  <span>RENDERER: SOFTWARE SVG · VALIDATION: OBJ REIMPORT + SILHOUETTE IoU</span>
</footer>
</div>
<script>
document.querySelectorAll('.view-switch button').forEach(function (button) {
  button.addEventListener('click', function () {
    var card = button.closest('.card');
    card.dataset.view = button.dataset.view;
  });
});
</script>
</body>
</html>`)
	return out.String()
}

func writeHeader(out *strings.Builder, page Page) {
	passLabel := "全部通过"
	if !page.Summary.AllPass {
		passLabel = "存在偏差"
	}
	fmt.Fprintf(out, `<div class="topbar">
  <a class="brand" href="#top">
    <span class="brand-mark">△</span>
    <span class="brand-copy"><strong>MESH / BOOLEAN</strong><span>COMPUTATIONAL GEOMETRY LAB</span></span>
  </a>
  <div class="status">
    <span>GO 1.27 · ZERO DEPENDENCIES</span>
    <span class="status-live">%s</span>
  </div>
</div>
<header class="hero" id="top">
  <div>
    <div class="kicker">CSG KERNEL / TRIANGLE SOUP / EXACT COPLANAR ROUTING</div>
    <h1>三角网格<br><span class="outline">布尔实验室</span></h1>
    <p class="hero-copy">
      对 <strong>并集 ∪、交集 ∩、差集 −</strong> 进行 BSP 裁剪。跨面片在交点处分裂，
      共面片按法线方向唯一路由，内部片被删除，输出壳体按连接关系与嵌套深度重新统一法线。
      下方 9 份结果均由同一套 Go 内核实时计算并渲染。
    </p>
    <div class="hero-tags">
      <span class="pill">COPLANAR SAFE</span>
      <span class="pill">INTERSECTION SPLIT</span>
      <span class="pill">CAVITY-AWARE NORMALS</span>
      <span class="pill">OBJ ROUNDTRIP</span>
    </div>
  </div>
  <aside class="panel hero-panel">
    <div class="panel-label">RUNTIME SNAPSHOT / %s</div>
    <div class="stat-grid">
      <div class="stat"><b>%02d</b><span>TEST SCENES</span></div>
      <div class="stat"><b>%02d</b><span>BOOLEAN RESULTS</span></div>
      <div class="stat"><b>%s</b><span>RESULT TRIANGLES</span></div>
      <div class="stat"><b class="green">%.3f%%</b><span>MIN SILHOUETTE IoU</span></div>
      <div class="stat"><b>%s</b><span>INTERIOR FACES</span></div>
      <div class="stat"><b class="green">%d/9</b><span>VALIDATION PASS</span></div>
    </div>
    <div class="pipeline">
      <div class="pipe-row"><i>01</i><span>平面分类</span><small>PLANE TEST</small></div>
      <div class="pipe-row"><i>02</i><span>交点分裂</span><small>SPLIT &amp; CLIP</small></div>
      <div class="pipe-row"><i>03</i><span>共面唯一化</span><small>COPLANAR ROUTE</small></div>
      <div class="pipe-row"><i>04</i><span>内部面删除</span><small>INSIDE CULL</small></div>
      <div class="pipe-row"><i>05</i><span>法线统一</span><small>SHELL ORIENT</small></div>
    </div>
  </aside>
</header>
`, passLabel, html.EscapeString(page.Summary.GeneratedAt), page.Summary.Cases,
		page.Summary.Operations, formatInt(page.Summary.Triangles),
		page.Summary.MinIOU*100, formatInt(page.Summary.InteriorFaces), countPass(page))
}

func writeLegend(out *strings.Builder) {
	out.WriteString(`<div class="legend">
  <div class="legend-left">
    <span class="legend-item"><i class="swatch union"></i>并集</span>
    <span class="legend-item"><i class="swatch intersection"></i>交集</span>
    <span class="legend-item"><i class="swatch difference"></i>差集</span>
    <span class="legend-item"><i class="swatch internal"></i>内部/裁剪参考面</span>
  </div>
  <span>ORTHO ISO VIEW · INTERIOR OVERLAY ENABLED</span>
</div>
`)
}

func writeCase(out *strings.Builder, view CaseView) {
	fmt.Fprintf(out, `<section class="case" id="%s">
  <div class="case-head">
    <div class="case-number">%s</div>
    <div>
      <div class="case-title"><h2>%s</h2><span>%s</span></div>
      <p class="case-desc">%s</p>
    </div>
    <div class="case-tags">`, html.EscapeString(view.ID), view.Number,
		html.EscapeString(view.Title), html.EscapeString(view.English),
		html.EscapeString(view.Description))
	for _, tag := range view.Tags {
		fmt.Fprintf(out, `<span class="tag">%s</span>`, html.EscapeString(tag))
	}
	out.WriteString(`</div></div><div class="inputs">`)
	for _, name := range view.Names {
		fmt.Fprintf(out, `<span class="input-chip"><b>%s</b></span>`, html.EscapeString(name))
	}
	out.WriteString(`<span class="input-chip"><b>OPERATOR</b> A ∪∩− B</span></div><div class="cards">`)
	for _, operation := range view.Operations {
		writeOperationCard(out, operation)
	}
	out.WriteString(`</div></section>`)
}

func writeOperationCard(out *strings.Builder, op OperationView) {
	statusClass, statusText := "pass", "PASS"
	if !op.Pass {
		statusClass, statusText = "", "CHECK"
	}
	fmt.Fprintf(out, `<article class="card %s" data-view="both">
  <div class="card-head">
    <div class="op-name">
      <span class="op-symbol">%s</span>
      <span class="op-title"><b>%s</b><small>%s</small></span>
    </div>
    <div class="op-volume"><b>%.4f</b><small>VOLUME</small></div>
  </div>
  <div class="stage">
    <div class="internal-badge">INTERIOR %s</div>
    <div class="result-render">%s</div>
    <div class="obj-render">%s</div>
    <div class="stage-meta"><b>OBJ IoU %.5f%%</b> · SIL %.1f%%</div>
  </div>
  <div class="view-switch">
    <button type="button" data-view="both">结果 + 内部</button>
    <button type="button" data-view="result">仅结果</button>
    <button type="button" data-view="internal">内部面</button>
    <button type="button" data-view="obj">OBJ 回读</button>
  </div>
  <div class="card-stats">
    <div class="metric"><b>%s</b><span>TRIANGLES</span></div>
    <div class="metric"><b>%s</b><span>INTERNAL</span></div>
    <div class="metric"><b>%.2e</b><span>VOL ERROR</span></div>
  </div>
  <div class="card-foot">
    <a href="%s" download>↓ EXPORT OBJ · %s FACES</a>
    <span class="%s">%s</span>
  </div>
</article>
`, op.Class, op.Symbol, html.EscapeString(op.Label), op.English,
		op.Volume, formatInt(op.InteriorTris), op.SVG, op.OBJSVG,
		op.IOU*100, op.Coverage*100, formatInt(op.Triangles),
		formatInt(op.InteriorTris), op.AbsError,
		html.EscapeString(op.OBJPath), formatInt(op.OBJFaces),
		statusClass, statusText)
}

func writeMethod(out *strings.Builder) {
	out.WriteString(`<div class="section-head">
  <h2>裁剪路径</h2>
  <p>布尔结果不是像素遮罩。每个三角面都经过平面分类、跨面求交和实体侧判定，再以 OBJ 重读后的轮廓作为最终一致性证据。</p>
</div>
<div class="method-grid">
  <div class="method"><i>STEP 01</i><h3>平面分类</h3><p>使用 EPS = 1e-7 将顶点划为前侧、后侧或共面，跨面片保留切分参数 t。</p></div>
  <div class="method"><i>STEP 02</i><h3>交点分裂</h3><p>边与切割平面求交，重新组成前、后凸多边形；曲面由连续三角形分裂逼近。</p></div>
  <div class="method"><i>STEP 03</i><h3>共面路由</h3><p>共面片按两侧法线点积分流，使并集只保留一层，交集不产生空洞。</p></div>
  <div class="method"><i>STEP 04</i><h3>内部面删除</h3><p>BSP 叶节点判定实体内外，删除被包围面片；诊断层保留其中片段并以红色叠显。</p></div>
  <div class="method"><i>STEP 05</i><h3>法线统一</h3><p>焊接接缝后沿共享边传播方向，再按壳体嵌套深度区分外表面与空腔表面。</p></div>
</div>
`)
}

func writeVerification(out *strings.Builder, page Page) {
	fmt.Fprintf(out, `<div class="section-head">
  <h2>验证矩阵</h2>
  <p>体积与解析期望值比较；OBJ 以 17 位有效数字导出并重读，在相同相机下重新栅格化轮廓。最小值 %.5f%%。</p>
</div>
<div class="verify-wrap">
<table>
  <thead><tr><th>SCENE</th><th>OPERATION</th><th>VOLUME</th><th>EXPECTED</th><th>ABS ERROR</th><th>TRIANGLES / OBJ</th><th>SILHOUETTE</th><th>CHECK</th></tr></thead>
  <tbody>
`, page.Summary.MinIOU*100)
	for _, view := range page.Cases {
		for _, op := range view.Operations {
			check := `<span class="check">PASS</span>`
			if !op.Pass {
				check = `<span style="color:var(--red)">CHECK</span>`
			}
			fmt.Fprintf(out, `<tr>
  <td>%s</td>
  <td><span class="op-dot %s"></span>%s</td>
  <td class="n">%.6f</td>
  <td class="n dim">%.6f</td>
  <td class="n">%.3e</td>
  <td class="n">%s / %s</td>
  <td class="n">%.5f%%</td>
  <td>%s</td>
</tr>`, html.EscapeString(view.Title), op.Class, html.EscapeString(op.Label),
				op.Volume, op.Expected, op.AbsError,
				formatInt(op.Triangles), formatInt(op.OBJFaces), op.IOU*100, check)
		}
	}
	out.WriteString(`</tbody></table></div>`)
}

func countPass(page Page) int {
	count := 0
	for _, view := range page.Cases {
		for _, op := range view.Operations {
			if op.Pass {
				count++
			}
		}
	}
	return count
}

func formatInt(value int) string {
	text := fmt.Sprintf("%d", value)
	if len(text) <= 3 {
		return text
	}
	var parts []string
	for len(text) > 3 {
		parts = append([]string{text[len(text)-3:]}, parts...)
		text = text[:len(text)-3]
	}
	parts = append([]string{text}, parts...)
	return strings.Join(parts, ",")
}
