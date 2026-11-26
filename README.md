# outline-lab：中英文轮廓 · 布点 · 行布局 · 描边预览

纯 Go 标准库实现（无第三方依赖、无需安装任何组件）。程序直接解析系统
TrueType 字体的 `cmap / glyf / loca / hmtx / head / hhea` 表，得到中英文
原始贝塞尔轮廓，再完成弧长布点、轮廓级排版换行、按真实墨迹边界的行布局、
SSAA 抗锯齿光栅化与字腔安全的粗体/描边预览，并输出原始轮廓、排版结果和
多张局部放大图。

## 运行

```bash
go run .                 # 生成全部产物到 artifacts/
go run . -out /tmp/out   # 也可指定输出目录
go test ./...            # 几何/排版/字腔不变量测试 + 端到端出图
```

使用的系统字体（无需安装，环境自带）：

- 拉丁/数字：`/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf`
- 中文：`/usr/share/fonts/truetype/droid/DroidSansFallbackFull.ttf`

## 产物清单（`artifacts/`）

- `01_original_outlines.png`：原始轮廓、在曲线上点/二次控制点/控制多边形。
- `02_outline_points.png`：每条轮廓按弧长均匀布点（含切向、法向、首点）。
- `03_layout_result.png`：排版结果（几何标注版）：基线、上升顶、下降底、
  首行字形包围盒、字距/行间隙尺寸、统计卡。
- `04_layout_clean.png`：无标注的干净排版输出（含安全粗体标题）。
- `05_stroke_preview.png`：填充 / 细描边 / 粗描边 / 字腔安全粗体对照。
- `06_zoom_linegap.png`：局部放大——上行下降部与下行上升部的正间隙。
- `07_zoom_tracking.png`：局部放大——字距 T 如何加入笔位推进。
- `08_zoom_counters.png`：局部放大——常规 / 采用的安全粗体 / 被拒绝的过粗。
- `09_zoom_points.png`：局部放大——字母 `g` 字腔处的等弧长布点与法向。
- `10_overview.png`：以上九图的总览拼图。
- `report.json`：布局坐标、逐行间隙、三项校验、粗体字腔决策。
- `outline_data.json`：样例字的 EM 归一化采样点与弧长误差（机器精度）。

## 关键正确性

- **字距在轮廓坐标里**：`x(n+1) = x(n) + advance(n) + tracking`，每个字形
  都有笔位与真实 `glyf` 包围盒，而不是只设置文本框属性。
- **行高保证上升部/下降部不互相覆盖**：相邻行的最小基线步距取
  `max(LineHeightEm·ppem, descent上一行 + minGap + ascent下一行)`，
  并用真实墨迹包围盒复核；`report.json` 中
  `ascender_descender_no_overlap=true`，所有行对间隙均为正。
- **粗体不吞字腔**：粗体 = 填充 ∪ 宽度 `2r` 的居中轮廓描边（单侧膨胀 `r`）。
  对常规轮廓泛洪找出封闭字腔，逐档增大 `r` 并复测字腔面积/腐蚀深度；
  不达标就逐字回退（如 `量→0.010em`、`国→0.025em`、`a→0.045em`），
  `r=0.11em` 的破坏性方案会被判定为 `FAIL` 且不用于任何正式产物。

## 代码地图

- `ttf.go` `ttf_cmap.go` `ttf_glyf.go`：TTF 表解析与简单/复合字形解码。
- `geom.go`：二次贝塞尔自适应展平、弧长表、等弧长采样、法向、适配变换。
- `scanline.go` `scene.go` `imageutil.go`：扫描线 even-odd 填充、圆头粗路径
  点印、超采样画布、最近邻放大与拼图。
- `layout.go`：分词、CJK/拉丁换行、笔位、墨迹边界与垂直行布局校验。
- `counters.go`：字腔泛洪检测与粗度安全回退。
- `scene_*.go` `zoom.go`：各张图片的构图。
- `report.go`：`report.json` / `outline_data.json`。
- `typography_test.go`：不变量测试。
