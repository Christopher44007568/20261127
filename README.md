# fillmask — 奇偶 / 非零填充规则、裁剪与嵌套子路径

纯 Go 标准库实现（无 `require`、无 `go.sum`、不安装任何组件）。对同一条自交
五角星路径分别应用 **EvenOdd（奇偶）** 与 **NonZero（非零环绕数）** 规则，
得到不同的孔洞；同时实现矩形/任意路径裁剪与三层嵌套子路径，输出填充掩码、
边界放大图、合成 PNG 和像素级 JSON 报告。

## 运行

```sh
go run . -out artifacts      # 或：go run main.go -out artifacts
go test ./...
go vet ./...
```

运行结束时所有像素检查必须为 `PASS`，否则进程以非零码退出。

## 产物（写入 `artifacts/`）

填充掩码（8-bit 灰度 PNG，覆盖=255，孔洞=0，只有 0/255 两种值）：

- `mask_star_evenodd.png` / `mask_star_nonzero.png`：同一条自交五角星，
  中心五边形在奇偶规则下是孔洞、在非零规则下环绕数为 2 被填充。
- `mask_nested_evenodd.png` / `mask_nested_nonzero.png`：外圆角矩形 +
  反向内孔矩形 + 正向中心菱形岛。
- `mask_clipped_rect_nonzero.png`：整数矩形裁剪后的嵌套路径。
- `mask_clipped_circle_evenodd.png`：任意路径（圆形裁剪区）裁剪五角星。

边界放大图（最近邻放大 + 设备像素网格）：

- `zoom_star_core_evenodd.png` / `zoom_star_core_nonzero.png`：同一裁剪框
  跨过中心五边形边界，直接对比“有孔 / 无孔”。
- `zoom_hole_edge.png`：嵌套边界，依次为填充区、孔洞（棋盘纸底）、蓝色内岛。
- `zoom_clip_edge.png`：裁剪接缝的黑—白—黑整像素条带。
- `boundary_zooms.png`：以上四张的 2×2 总览。

合成图与报告：

- `composite.png`：六宫格合成 PNG（两种规则、嵌套、矩形裁剪、路径裁剪）。
- `report.json`：每项像素不变量的检查结果与统计。

## 两个关键正确性约束

**孔洞不能被主轮廓覆盖。** 一条 `Path` 的全部子路径在 *同一次* 分类中累积
射线交点的奇偶/带符号环绕数（见 `demo/mask.go` 的 `ClassifyLoops`，半开
`<= / >` 跨越约定），先得到唯一的整体掩码，再由 `PaintMask` 只绘制一遍。
外轮廓与内孔不是“分别填充再叠加”，因此主轮廓没有机会事后盖住孔洞。验证
`holes_not_covered_by_main_contour` 逐像素比较：所有“仅外轮廓覆盖、但统一
掩码为 0”的孔洞像素，合成后必须与纸底逐字节相同（本次运行为 15264 个像素、
0 泄漏）。

**裁剪边缘不能有半像素缝隙。** 掩码是二值的（不存在 1..254 的覆盖值），矩形
裁剪是整像素 AND：`Min` 含、`Max` 排他，边界像素要么整列 255 要么整列 0，
结构上不可能出现半覆盖缝。浮点裁剪框用 `SnapClipRectOutward` 向外取整
（`floor(Min)`、`ceil(Max)`），保证任何被请求区域碰到的像素都不会被削成
半像素。`clip_edges_have_no_half_pixel_gap` 对裁剪框四条边逐行/逐列检查：
形状跨过接缝时边界像素必须全满、不得出现“覆盖-0-覆盖”的夹心断缝。

## 目录

- `main.go`：薄命令行入口。
- `demo/geom.go`：路径构建、二次/三次贝塞尔自适应展平、方向归一化与基本图形。
- `demo/mask.go`：`Mask`、`EvenOdd`/`NonZero` 与环绕数分类。
- `demo/clip.go`：矩形/掩码/路径裁剪、浮点向外对齐、四条接缝检查。
- `demo/render.go`、`demo/pipeline.go`：纸底、掩码合成、六宫格与报告编排。
- `demo/zoom.go`：最近邻放大、像素网格与标注。
- `demo/font.go`、`demo/glyphs.go`：内置 5×7 位图字体（无需外部字体模块）。
- `demo/scenes.go`：五角星与三层嵌套场景坐标。
- `demo/verify.go`、`demo/fillmask_test.go`：像素级校验与端到端测试。
