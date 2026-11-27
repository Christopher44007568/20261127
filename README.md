# dirtycanvas：大画布拖动的脏矩形渲染

只用 Go 标准库实现的脏矩形（dirty rectangle）重绘演示：在 1280×800 的大画布上
生成静态背景、脚本化拖动三个图形，并使用可复用的缓存文字纹理。每一帧都只重绘
“旧包围盒 ∪ 新包围盒”合并出的脏矩形，结果与整画布全量重绘逐像素一致。

## 运行

```sh
go run .
go test ./...
```

常用参数：

```sh
go run . -out artifacts -width 1280 -height 800 -frames 24
```

程序会逐帧把增量结果与全量参考渲染做像素比较，任何不一致都会以非零状态退出，
并在标准输出打印 `PIXEL_PERFECT=true ...`。

## 产物（默认写入 `artifacts/`，PPM 为 P6 二进制格式）

| 文件 | 内容 |
| --- | --- |
| `dirty_region.ppm` / `dirty.ppm` / `dirty_rect.ppm` / `dirty_map.ppm` | 最后一帧的脏区图：画面压暗，红色填充 + 黄色描边为实际重绘矩形 |
| `dirty_mask.ppm` | 脏区黑白蒙版：白色为参与重绘的像素 |
| `incremental_result.ppm` / `incremental.ppm` | 脏矩形增量渲染的最终画面 |
| `full_reference.ppm` / `reference.ppm` / `full.ppm` | 同一场景的全量重绘参考 |
| `diff.ppm` | 增量与参考的差异放大图；完全一致时为纯黑 |
| `font_sheet.ppm` / `.png` | 内置 5×7 点阵字体字样表 |
| `report.txt` / `report.json` | 每帧脏区数量、脏区占比与像素一致性报告 |

主要 PPM（脏区图、蒙版、增量、参考、字体表）另附同名 `.png` 预览（标准库 `image/png`）。
可用 `-out` 自定义目录；工作区里的 `outputs/` 是用 `go run . -out outputs` 生成的同内容副本。
`incremental_result.ppm` 与 `full_reference.ppm` 字节级完全相同。

## 为什么能保证逐像素一致

- **旧脏区不会漏清**：对象变化时，脏区同时加入“上一帧包围盒”和“当前包围盒”
  （`DirtyTracker.Collect` / `Region.AddChange`）。移动图形的残影区域一定在脏区内。
- **先恢复静态背景再按同一 z-order 重绘全部对象**：每个合并后的脏矩形内，
  `PaintScene` 先从不可变背景缓冲拷贝对应区域，再用与全量渲染完全相同的绘制函数
  重画所有对象，并统一裁剪到该矩形。
- **脏矩形先合并为互不相交的集合**（`MergeRects`），避免重叠区域被 alpha 混合两次。
- **所有图元都走同一个带 clip 的 `Framebuffer`**，脏矩形外的像素在 API 层面不可写。
- **文字缓存更新先清空整块复用表面**（`CachedText.SetText` 中的 `clear(surf.Pix)`），
  新串变短时上一串字符的残留纹素全部为 0；同时标签的旧/新包围盒都进脏区，
  画布上也不会留下“尾巴”。

## 测试

```sh
go test ./...
```

- `TestDemoRunPixelPerfectAndWritesPPM`：完整模拟 + PPM 落盘回读，增量与参考一致。
- `TestOldDirtyPixelsAreRestored`：图形移走后，旧中心像素恢复为静态背景色。
- `TestShrinkingTextLeavesNoRemnants`：长串改短串后，旧尾部像素与全量参考一致。
- `TestCachedTextShrinkLeavesNoRemnants`：直接检查缓存表面新宽度之外 alpha 全为 0，
  且暴露的 Sprite 与全新渲染结果一致。
- `TestFuzzRandomMovesAndText`：400 步随机移动/随机文案/中途新增对象，
  每一步都要求增量与全量逐像素相同。
- 另有矩形合并、PPM 往返编解码、字体度量一致性测试。

## 代码结构

- `geom.go`：整数矩形、交集/并集、重叠矩形合并。
- `framebuffer.go`：RGB 帧缓冲、裁剪、矩形/圆盘/三角形/菱形图元、Sprite alpha blit。
- `ppm.go`：P6/P3 PPM 读写与 PNG 输出。
- `font.go`、`font_glyphs.go`、`font_glyphs_extra.go`：内置 5×7 点阵字体。
- `textcache.go`：会清空旧内容的单槽复用文字缓存。
- `dirty.go`：脏区集合与按版本/包围盒追踪的 `DirtyTracker`。
- `background.go`：程序化静态背景（渐变、网格、标题）。
- `scene.go`：图形、文字面板、场景与统一的 `PaintScene`/`RenderFull`。
- `dirtymap.go`：脏区图、蒙版、差异图、字体表。
- `simulate.go`：脚本化拖动、逐帧校验、报告与产物落盘（`Run`）。
- `main.go`：命令行入口。
