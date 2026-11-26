package report

import (
	"fmt"
	"os"
	"strings"

	"featurelines/internal/pipeline"
)

// WriteMarkdown emits the Chinese acceptance report. imageRel is the relative
// path from the markdown file to the rendered image directory.
func WriteMarkdown(path, imageRel string, outs []*pipeline.Output) error {
	if err := os.MkdirAll(dirOf(path), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	w := func(s string) { b.WriteString(s) }

	w("# 曲面特征线追踪与端点误差验收报告\n\n")
	w("纯 Go 标准库实现，无第三方依赖。流程：程序化验收模型 → 离散曲率/法线夹角信号 → ")
	w("双阈值迟滞筛选 → 拓扑成链 → 单顶点缺口桥接与闭环 → 覆盖率/端点误差评估 → 软件渲染出图。\n\n")

	allPass := true
	for _, o := range outs {
		allPass = allPass && o.Metrics.Pass
	}
	fmt.Fprintf(&b, "**总体结论：%s**\n\n", map[bool]string{true: "全部验收项通过 ✅", false: "存在未通过项 ❌"}[allPass])

	w("## 方法要点\n\n")
	w("- **法线夹角 θ**：每条邻接面边计算二面角；θ < 5° 视为 C1 相切并直接否决，曲率差再大也不产生特征线（保护相切曲面）。\n")
	w("- **曲率差 Δκ**：余切 Laplace 估计顶点曲率，按内部顶点 98 分位归一化；用边两侧对顶点的曲率跳变刻画跨边曲率差。\n")
	w("- **迟滞追踪**：θ≥22° 为强边；8°≤θ<22° 且 Δκ≥0.85 为强曲率确认边；8° 以上、Δκ≥0.30 为弱边，弱边只有与强边连通才保留。\n")
	w("- **抗单顶点失败**：失败顶点使相邻两条边失去证据时，沿拓扑搜索 ≤2 跳、用切向连续性配对两端并桥接；同链两端配对即闭合成环。连续特征不会因一个顶点失败而整段消失。\n\n")

	w("## 验收结果\n\n")
	w("| 场景 | 覆盖率 | 闭环 | 特征端点数/最大误差 | 桥接 | 朴素高阈值基线 | 结论 |\n")
	w("|---|---|---|---|---|---|---|\n")
	for _, o := range outs {
		m := o.Metrics
		cov := fmt.Sprintf("%.1f%%", m.Coverage*100)
		closure := "开放"
		if m.ClosedLoop {
			closure = "闭合"
		}
		ends := fmt.Sprintf("%d / %.2f mm", m.EndpointCount, m.MaxEndpointError)
		naive := fmt.Sprintf("%d 段, %d 个假端点", m.NaiveSegments, m.NaiveEndpoints)
		if o.Case.Kind == "tangent" {
			ends = fmt.Sprintf("误检长度 %.2f mm", m.FalseFeatureLength)
			naive = fmt.Sprintf("纯曲率差误报 %.0f mm", m.CurvOnlySeamLength)
			cov, closure = "—（负例）", "—"
		}
		verdict := "PASS"
		if !m.Pass {
			verdict = "FAIL"
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %d | %s | %s |\n",
			o.Case.Title, cov, closure, ends, m.BridgeCount, naive, verdict)
	}
	w("\n")

	w("## 三项渲染总览\n\n")
	fmt.Fprintf(&b, "![总览](%s/00_overview.png)\n\n", imageRel)

	titles := map[string]string{
		"hole":    "闭合圆孔",
		"sharp":   "开放锐边",
		"tangent": "相切曲面",
	}
	for _, o := range outs {
		id := o.Case.ID
		fmt.Fprintf(&b, "## %s — %s\n\n", titles[id], o.Case.Title)
		for _, chk := range o.Metrics.Checks {
			mark := "✅"
			if !chk.OK {
				mark = "❌"
			}
			fmt.Fprintf(&b, "- %s `%s` — %s\n", mark, chk.Name, chk.Detail)
		}
		w("\n")
		fmt.Fprintf(&b, "![模型](%s/%s_1_model.png)\n\n", imageRel, id)
		fmt.Fprintf(&b, "![叠加](%s/%s_2_overlay.png)\n\n", imageRel, id)
		fmt.Fprintf(&b, "![端点误差](%s/%s_3_endpoint_errors.png)\n\n", imageRel, id)
	}

	w("## 运行方式\n\n")
	w("```bash\n")
	w("go run .              # 生成全部 PNG / metrics.json / features.obj / 本报告\n")
	w("go test ./...         # 验收与桥接单元测试\n")
	w("go run ./cmd/check    # 控制台打印指标\n")
	w("```\n\n")
	w("产物目录：`artifacts/render/`（九张单图、三张分类长图 `01_original_models.png`、")
	w("`02_feature_overlay.png`、`03_endpoint_errors.png` 与 `00_overview.png`），")
	w("`artifacts/metrics.json`，`artifacts/features.obj`。\n")

	return os.WriteFile(path, []byte(b.String()), 0o644)
}
