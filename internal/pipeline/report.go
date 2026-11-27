package pipeline

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"materiallod/material"
)

func writeReportMarkdown(dir string, art *Artifacts) error {
	var b strings.Builder
	b.WriteString("# 金属 / 布 / 塑料的三档光照近似\n\n")
	b.WriteString("Go 标准库实现，外部依赖为零。所有材质共用同一参数集；高、中、低三档只增减光照项，不替换主色和粗糙区间。\n\n")

	b.WriteString("## 渲染产物\n\n")
	b.WriteString("- `spheres.png`：三行材质（金属、布、塑料）× 三列档位（低、中、高）材质球。\n")
	b.WriteString("- `view_sweep.png`：固定法线、视角 0°–90° 的显示亮度扫掠。\n")
	b.WriteString("- `brightness_diff.png`：对齐材质球相对高档的绝对亮度差热力图。\n")
	b.WriteString("- `boundary_continuity.png`：连续 LOD 扫描下的误差与步间跳变。\n")
	b.WriteString("- `view_sweep.csv`、`brightness_difference.csv`、`boundary_errors.csv`：原始采样数据。\n")
	b.WriteString("- `continuity_report.json`、`brightness_stats.json`、`materials.json`：结构化报告。\n\n")

	writeParameterTable(&b, art.Materials)
	writeBrightnessMarkdown(&b, art.Brightness)
	writeBoundaryMarkdown(&b, art.Continuity)
	writePreservationMarkdown(&b, art.Continuity.Preservation)

	b.WriteString("## 复现\n\n")
	b.WriteString("```bash\n")
	b.WriteString("go test ./...\n")
	b.WriteString("go run ./cmd/material-lod -out artifacts/material-lod\n")
	b.WriteString("```\n")

	path := filepath.Join(dir, "report.md")
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	return nil
}

func writeParameterTable(b *strings.Builder, materials []material.Material) {
	b.WriteString("## 统一参数集\n\n")
	b.WriteString("| 材质 | 主色 | Metallic | Roughness | 粗糙区间 | Sheen | Clearcoat |\n")
	b.WriteString("| --- | --- | ---: | ---: | --- | ---: | ---: |\n")
	for _, m := range materials {
		srgb := m.BaseColor.ToSRGB()
		hex := fmt.Sprintf("#%02X%02X%02X",
			byte(math.Round(srgb.R*255)), byte(math.Round(srgb.G*255)), byte(math.Round(srgb.B*255)))
		b.WriteString(fmt.Sprintf("| `%s` | `%s` | %.2f | %.2f | [%.2f, %.2f] | %.2f | %.2f |\n",
			m.ID, hex, m.Metallic, m.Roughness, m.RoughnessMin, m.RoughnessMax,
			m.SheenWeight, m.ClearcoatWeight))
	}
	b.WriteString("\n低档 `LowUniforms` 只保留主色、金属度、名义粗糙度和粗糙区间；省略 GGX、Smith、Schlick、环境反射、Sheen 与 Clearcoat 项。\n\n")
}

func writeBrightnessMarkdown(b *strings.Builder, stats []material.BrightnessStats) {
	b.WriteString("## 亮度差（材质球，相对高档）\n\n")
	b.WriteString("| 材质 | 对比 | 像素数 | MAE | RMSE | Max | 平均有符号差 |\n")
	b.WriteString("| --- | --- | ---: | ---: | ---: | ---: | ---: |\n")
	for _, s := range stats {
		b.WriteString(fmt.Sprintf("| `%s` | `%s` | %d | %.4f | %.4f | %.4f | %+.4f |\n",
			s.MaterialID, s.Compare, s.Pixels, s.MAE, s.RMSE, s.MaxAbs, s.MeanSigned))
	}
	b.WriteString("\n误差基于共享色调映射和 sRGB 转换后的亮度；热力图满刻度为 `0.28`。\n\n")
}

func writeBoundaryMarkdown(b *strings.Builder, report material.ContinuityReport) {
	b.WriteString("## 切换边界连续性\n\n")
	b.WriteString(fmt.Sprintf("连续 LOD 坐标为 `0=低`、`1=中`、`2=高`；硬切换边界是 `0.5` 与 `1.5`，crossfade 半宽为 `%.2f`，扫描步长 `%.2f`。步间亮度跳变阈值为 `%.3f`。\n\n",
		report.TransitionWidth, report.LODStep, report.JumpThreshold))
	b.WriteString("| 边界 | 过渡 | 相邻档 MAE | RMSE | Max | 连续扫描最大步跳 | 结论 |\n")
	b.WriteString("| --- | --- | ---: | ---: | ---: | ---: | --- |\n")
	for _, boundary := range report.Boundaries {
		status := "**PASS**"
		if !boundary.Passed {
			status = "**FAIL**"
		}
		b.WriteString(fmt.Sprintf("| %.1f | `%s` | %.4f | %.4f | %.4f | %.5f | %s |\n",
			boundary.BoundaryLOD, boundary.Transition, boundary.PairMAE, boundary.PairRMSE,
			boundary.PairMax, boundary.ContinuousMaxStep, status))
	}
	b.WriteString("\n边界误差在固定法线的 91 点视角亮度序列上计算。`Pair*` 是不做 crossfade 时硬切换会产生的跨材质汇总亮度差；`连续扫描最大步跳` 是实际连续混合路径在细粒度扫描中的最大相邻视角采样差。完整逐 LOD、逐材质数据见 `boundary_errors.csv`。\n\n")
}

func writePreservationMarkdown(b *strings.Builder, rows []material.PreservationReport) {
	b.WriteString("## 主色与粗糙区间保持\n\n")
	b.WriteString("| 材质 | 主色 RGB 差 | 色相角差 | 粗糙度下限差 | 上限差 | 结论 |\n")
	b.WriteString("| --- | ---: | ---: | ---: | ---: | --- |\n")
	for _, p := range rows {
		status := "**PASS**"
		if !p.ColorPreserved || !p.RangePreserved {
			status = "**FAIL**"
		}
		b.WriteString(fmt.Sprintf("| `%s` | %.2e | %.3f° | %.2e | %.2e | %s |\n",
			p.MaterialID, p.BaseColorDelta, p.HueDeltaDeg,
			p.RoughnessMinDelta, p.RoughnessMaxDelta, status))
	}
	b.WriteString("\n")
}
