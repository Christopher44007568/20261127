# 矢量场景：SVG 导出 / 双端光栅化 / 结构差分（纯 Go）

同一个矢量场景（`BuildScene`，960×600）依次完成：

1. 导出为**可编辑 SVG** `artifacts/scene.svg`：每个原对象保留稳定 `id`，
   填充、描边、`linearGradient`/`radialGradient`、`clipPath` 全部是原生 SVG 语义，
   不嵌入任何位图；
2. 同一份 SVG 交给无头 **Chrome/Edge（含 WSL 互操作调用 Windows 浏览器）**
   生成浏览器参考图 `artifacts/browser_reference.png`；
3. 用**仅标准库的 Go 光栅化器**（路径自适应展平 + 4× SSAA + 盒式降采样，
   nonzero 填充、圆头/圆角描边、渐变着色、裁剪栈、ID 缓冲）生成
   `artifacts/own.png`；
4. 逐像素对齐比较，按 ID 缓冲把错误像素归属到**原对象 ID**，输出热图、
   ID 分区图、带 ID 标注的错误图层和结构差分总览图；
5. 用 `encoding/xml` **重新载入 SVG**，审计元素树并再次光栅化
   `artifacts/reloaded.png`，证明所有路径仍是彼此独立的可编辑矢量元素，
   而不是合并成一张位图。

模块零第三方依赖，不需要 `go get`，也不安装任何系统组件。

## 运行

```sh
go run .                 # 全量生成到 artifacts/（默认 -ss 4）
go run . -out out -ss 2  # 更快的低采样草稿
go run . -skip-browser   # 不调用浏览器（参考图回退为自身渲染）
go test -count=1 ./...   # SVG 语义 / 重载 / 裁剪 / 故障归属测试
```

浏览器按 `CHROME_PATH` → Windows 侧 `chrome.exe`/`msedge.exe`
（`/mnt/c/Program Files...`，经 WSL 互操作）→ PATH 中的
`google-chrome`/`chromium`/`msedge` 顺序发现。

## 产物导航

| 文件 | 内容 |
| --- | --- |
| `artifacts/scene.svg` | 可编辑矢量场景（20 个对象 ID、19 条独立 path、4 个渐变、1 个 clipPath，0 个 `<image>`） |
| `artifacts/browser_reference.png`（别名 `browser_ref.png`） | 无头 Chrome 对同一 SVG 的参考光栅化，960×600 |
| `artifacts/own.png`（别名 `self.png`、`own_render.png`） | 纯 Go SSAA 自身 PNG |
| `artifacts/reloaded.png` | 解析回 `scene.svg` 后再次光栅化，用于往返一致性核对 |
| `artifacts/diff.png`（别名 `structural_diff.png`） | 结构差分总览：参考/自身/热图/ID 图/错误图层 + 按对象 ID 的结构表 + 重载审计 |
| `artifacts/error_layer.png` | 错误图层：暗化底图、红色错误像素、对象包围盒与原对象 ID 标签 |
| `artifacts/fault_demo.png` | 注入故障（径向渐变→错误平涂）后，差分正确定位并标注 `obj-radial-gradient` |
| `artifacts/heatmap.png` | 放大 3 倍的像素差异热图 |
| `artifacts/id_map.png` | 逐像素对象 ID 分区图（每个对象一种颜色，证明对象彼此分离） |
| `artifacts/flattened_bad.svg` | 反例基线：把整帧嵌成 1 张 base64 `<image>`；审计必须判定为“已合并位图” |
| `artifacts/diff_report.json` | 全部指标、逐 ID 错误统计、8 项检查、重载审计与浏览器信息 |

## 本次运行结论

- 重载后 ID 零丢失：20/20；独立 path 19 条，与源场景可绘制对象数一致；
- `gradient=4`、`clipPath=1`、`<image>=0`，填充/描边/渐变/裁剪语义全部保留；
- `own.png` 与 `reloaded.png` 的平均像素差为 `0`（坐标短表示精确往返）；
- 自身 PNG 对浏览器参考：平均色差约 `0.24/255`，超阈值错误像素约 `0.03%`，
  残余差异仅为边缘抗锯齿；
- 注入故障归属：错误面积约 26k 像素，首要责任 ID 正是被注入的
  `obj-radial-gradient`；
- `flattened_bad.svg` 被审计识别为 `image=1 path=0` 的合并位图反例。

以上数字以 `artifacts/diff_report.json` 的实际内容为准。

## 代码结构

- `scene.go`：场景/对象/画刷/渐变/裁剪模型与演示场景；
- `geom.go`、`pathparse.go`：路径命令、自适应展平、包围盒、inside/描边距离与 SVG `d` 解析；
- `svgexport.go`、`svgparse.go`：SVG 导出与重载审计（`encoding/xml`）；
- `raster.go`：纯 Go SSAA 光栅化器与对象 ID 缓冲；
- `browser.go`：Chrome/Edge 无头参考图捕获（含 WSL 路径处理）；
- `diff.go`、`errorlayer.go`、`board.go`、`board_table.go`：对齐、差分、ID 归属、错误图层与结构差分图；
- `font.go`、`font_lower.go`、`font_punct.go`：内置 5×7 位图字体；
- `flatten.go`：“全部合并成一张位图”的反例 SVG；
- `main.go`、`pipeline.go`、`report.go`：流水线、检查项与 JSON 报告；
- `vector_test.go`：语义与归属的回归测试。
