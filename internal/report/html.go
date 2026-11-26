package report

import (
	"fmt"
	"strings"
)

func IndexHTML(b *Bundle) string {
	var h strings.Builder
	h.WriteString(`<!doctype html><html lang="zh-CN"><head><meta charset="utf-8">`)
	h.WriteString(`<meta name="viewport" content="width=device-width,initial-scale=1">`)
	h.WriteString(`<title>骨骼拓扑层次包围盒报告</title><style>`)
	h.WriteString(`:root{color-scheme:light}body{margin:0;background:#eef2ff;font-family:ui-sans-serif,system-ui,"Noto Sans CJK SC","Microsoft YaHei",sans-serif;color:#0f172a}.wrap{max-width:1180px;margin:0 auto;padding:28px}.hero{background:linear-gradient(135deg,#0f172a,#1d4ed8);color:#fff;border-radius:22px;padding:28px 32px;box-shadow:0 18px 50px #1e293b33}.hero h1{margin:0 0 10px;font-size:30px}.hero p{margin:6px 0;color:#dbeafe;line-height:1.6}.cards{display:grid;grid-template-columns:repeat(4,1fr);gap:14px;margin:20px 0}.card{background:#fff;border-radius:16px;padding:16px;box-shadow:0 8px 24px #0f172a12}.card .k{font-size:12px;color:#64748b}.card .v{font-size:24px;font-weight:800;margin-top:6px}.ok{color:#16a34a}.bad{color:#dc2626}section{background:#fff;border-radius:20px;padding:20px;margin:18px 0;box-shadow:0 8px 24px #0f172a10}h2{margin:0 0 14px}img{width:100%;height:auto;border:1px solid #e2e8f0;border-radius:14px;background:#f8fafc}table{border-collapse:collapse;width:100%;font-size:13px}th,td{border-bottom:1px solid #e2e8f0;padding:8px 10px;text-align:left}th{background:#f8fafc;font-size:12px;color:#475569}code,a{color:#1d4ed8}.links a{display:inline-block;margin:4px 12px 4px 0}.note{color:#64748b;font-size:13px;line-height:1.7}.grid2{display:grid;grid-template-columns:1fr 1fr;gap:16px}@media(max-width:900px){.cards,.grid2{grid-template-columns:1fr}}</style></head><body><div class="wrap">`)

	h.WriteString(`<div class="hero"><h1>按骨骼拓扑增量更新的层次包围盒</h1>`)
	fmt.Fprintf(&h, `<p>Go 标准库实现 · %s · %s · 无第三方包</p>`, b.GoVersion, b.GeneratedAt)
	h.WriteString(`<p>持久化骨骼节点只在首次构造时建树叶；姿态变化时向下传播脏矩阵、向上合并脏包围盒。三姿态的全部角色顶点均被覆盖。</p></div>`)

	h.WriteString(`<div class="cards">`)
	covered := 0
	for _, p := range b.Poses {
		if p.Coverage.Covered {
			covered++
		}
	}
	fmt.Fprintf(&h, `<div class="card"><div class="k">姿态覆盖</div><div class="v ok">%d/%d</div></div>`, covered, len(b.Poses))
	fmt.Fprintf(&h, `<div class="card"><div class="k">角色顶点</div><div class="v">%d</div></div>`, b.Poses[0].Coverage.VertexCount)
	fmt.Fprintf(&h, `<div class="card"><div class="k">骨骼节点</div><div class="v">%d</div></div>`, b.IncrementalTail.NodeCount)
	fmt.Fprintf(&h, `<div class="card"><div class="k">摆尾增量盒更新</div><div class="v">%d–%d</div></div>`, b.IncrementalTail.MinBoundsUpdates, b.IncrementalTail.MaxBoundsUpdates)
	h.WriteString(`</div>`)

	h.WriteString(`<section><h2>三姿态角色叠加盒</h2><p class="note">彩色盒为按深度着色的骨骼层次 AABB；红色虚线盒是“绑定整盒只跟随根节点”的对照，红点为实际漏出顶点。可在 SVG 中悬停查看每盒体积。</p>`)
	for _, p := range b.Poses {
		file := b.Files["overlay_"+p.Name]
		fmt.Fprintf(&h, `<h3>%s · <code>%s</code></h3><img src="%s" alt="%s overlay">`, poseTitle(p.Name), p.Name, file, p.Name)
	}
	h.WriteString(`</section>`)

	h.WriteString(`<section><h2>视锥剔除结果</h2><div class="grid2"><img src="culling.svg" alt="culling tree"></div>`)
	h.WriteString(`<table><thead><tr><th>姿态</th><th>视锥测试</th><th>绘制骨节点</th><th>剔除骨节点</th><th>节省测试</th></tr></thead><tbody>`)
	for _, c := range b.Culling {
		fmt.Fprintf(&h, `<tr><td>%s</td><td>%d / %d</td><td>%d</td><td class="bad">%d</td><td class="ok">%d</td></tr>`,
			poseTitle(c.Pose), c.Stats.FrustumTests, c.Stats.TotalNodes, c.Stats.VisibleBones, c.Stats.CulledBones, c.Stats.SavedFrustumTests)
	}
	h.WriteString(`</tbody></table><p class="note">规则：任一平面完全在外则剪枝整棵子树；所有平面完全在内则接受整棵子树；否则递归子节点。完整骨骼列表和相机参数见 <code>culling.json</code>。</p></section>`)

	h.WriteString(`<section><h2>逐层体积图</h2><img src="volume_layers.svg" alt="volume by hierarchy layer">`)
	h.WriteString(`<p class="note">机器可读数据：<a href="volume_layers.csv"><code>volume_layers.csv</code></a>。JSON 中同时记录每层 <code>volumeSum</code> 与同层盒并集体积 <code>unionVolume</code>。</p></section>`)

	h.WriteString(`<section><h2>增量更新证据</h2><table><thead><tr><th>姿态/帧</th><th>矩阵更新</th><th>包围盒更新</th><th>结构重建</th><th>覆盖</th><th>仅根盒漏出顶点</th></tr></thead><tbody>`)
	for _, p := range b.Poses {
		fmt.Fprintf(&h, `<tr><td>%s完整切姿</td><td>%d</td><td>%d</td><td>%d</td><td class="ok">%v</td><td class="bad">%d</td></tr>`,
			poseTitle(p.Name), p.Stats.MatrixUpdates, p.Stats.BoundsUpdates, p.Stats.StructuralRebuilds, p.Coverage.Covered, p.RootOnly.Violations)
	}
	fmt.Fprintf(&h, `<tr><td>连续摆尾 %d 帧（每帧范围，详见 <a href="update_trace.csv"><code>update_trace.csv</code></a>）</td><td>%v</td><td>%d–%d</td><td>%d</td><td>—</td><td>—</td></tr>`,
		b.IncrementalTail.Frames, b.IncrementalTail.MatrixUpdates, b.IncrementalTail.MinBoundsUpdates, b.IncrementalTail.MaxBoundsUpdates, b.IncrementalTail.StructuralRebuild)
	h.WriteString(`</tbody></table><p class="note">完整切姿会更新所有当前姿态矩阵和世界 AABB，但节点数组、父子关系和盒对象均为首次构造时分配；连续摆尾只触碰尾链与根祖先。单元测试 <code>TestLeafChangeUpdatesOnlyAncestorBounds</code> 固定了该行为。</p></section>`)

	h.WriteString(`<section class="links"><h2>数据与复现</h2>`)
	h.WriteString(`<a href="summary.json"><code>summary.json</code></a>`)
	h.WriteString(`<a href="result.json"><code>result.json</code></a>`)
	h.WriteString(`<a href="culling.json"><code>culling.json</code></a>`)
	h.WriteString(`<a href="volume_layers.csv"><code>volume_layers.csv</code></a>`)
	h.WriteString(`<a href="update_trace.csv"><code>update_trace.csv</code></a>`)
	h.WriteString(`<p class="note">运行 <code>go test ./...</code> 验证覆盖与增量更新；运行 <code>go run ./cmd/characterbounds -out artifacts</code> 重新生成全部图表。</p></section>`)

	h.WriteString(`</div></body></html>`)
	return h.String()
}
