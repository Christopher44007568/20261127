# 描边几何与重叠诊断

纯 Go 标准库实现的二维描边三角剖分示例，覆盖：

- `miter`、`round`、`bevel` 三种 join；
- `butt`、`round`、`square` 三种 cap；
- 奇偶虚线模式、`DashOffset`、闭合轮廓跨起点合并；
- 重复点、零长度开放/闭合子路径；
- miter limit 与近 180° 回折的安全处理；
- 仿射变换下的非均匀线宽规则；
- 纯填充图、4x4 抗锯齿图、真实三角形交集高亮图。

没有第三方依赖。

## 运行

```bash
go test ./...
go run ./cmd/stroke-demo
```

演示命令会生成：

- `artifacts/stroke-fill.png`：像素中心判定的纯填充三角形图；
- `artifacts/stroke-aa.png`：每像素 4x4 子采样并集，抗锯齿图；
- `artifacts/stroke-overlap.png`：只把具有正面积交集的三角形对染红。

## 基本用法

```go
p := stroke.NewPath()
p.MoveTo(stroke.Point{X: 20, Y: 100})
p.LineTo(stroke.Point{X: 120, Y: 20})
p.LineTo(stroke.Point{X: 220, Y: 100})

mesh := stroke.Stroke(*p, stroke.Style{
    Width:      12,
    Join:       stroke.JoinMiter,
    Cap:        stroke.CapRound,
    MiterLimit: 4,
    Dashes:     []float64{18, 10},
    DashOffset: 3,
})

diag := stroke.RenderDiagnostics(mesh, stroke.RasterOptions{
    Width:      800,
    Height:     500,
    Transform:  stroke.FitTransform(mesh.Bounds(), 800, 500, 32),
    Samples:    4,
    Background: color.NRGBA{R: 250, G: 248, B: 243, A: 255},
    Foreground: color.NRGBA{R: 23, G: 42, B: 58, A: 255},
    Overlap:    color.NRGBA{R: 220, G: 38, B: 38, A: 255},
})
```

`diag.Fill`、`diag.AntiAliased`、`diag.Overlaps` 都是 `*image.NRGBA`。
`diag.OverlapPairs` 返回发生正面积重叠的三角形索引对。

## Join 与 miter limit

相邻两段先求内外偏移线交点。miter 长度比按
`1 / cos(turnAngle/2)` 计算；超过 `MiterLimit` 时退化为 bevel，绝不继续
向外拉长。`MiterLimit <= 0` 时使用默认值 `4`。

转角达到 135° 时会直接切成两个带安全端帽的开放段，而不是求近乎平行的
超远偏移交点。这样既避免无限尖刺，也避免内侧三角形穿过相邻路径。

round join/cap 的弧线按 `ArcTolerance` 离散，默认最大弦高为 `0.05`
用户单位，并强制至少 4 段和对称采样，因此整圆能命中四个轴向极值点。

## 虚线规则

`Dashes` 是 `on/off` 交替长度。奇数项会复制一份变成偶数项；负值和非有限
值按 0 处理。`DashOffset` 表示每个子路径起点在模式中向前推进的距离。

虚线沿中心线弧长切分。切分后的每一段保留内部拐点的 join，只在真正的
断点使用 cap。闭合子路径若起点和终点都处于 `on` 状态，会合并为同一段，
不会在接缝处多出两个 cap。

如果虚线端点在某个拐点前不足内侧偏移所需距离，会对该端点做短桩裁剪：
让虚线段从拐点直接开始/结束，以端帽替代会重叠的 join 扇区。该规则针对
短虚线与锐角同时出现的情况，避免端帽三角形插进 join 三角形。

## 零长度子路径

- `CapButt`：不产生几何；
- `CapRound`：产生整圆扇形；
- `CapSquare`：产生以点为中心、边长为 `Width` 的方块。

连续的重复点会被去除，不会生成退化三角形。零长度子路径若处于虚线的
`off` 阶段，也不会产生点。

## 非均匀缩放

`Affine` 使用列向量形式：

```text
x' = A*x + C*y + E
y' = B*x + D*y + F
```

`StrokeTransformed` 的 `WidthPolicy` 明确定义线宽如何变化：

| Policy | 规则 |
| --- | --- |
| `WidthFollowTransform` | 先在局部空间描边，再变换网格。圆截面会变成椭圆，横向和纵向线宽分别变化 |
| `WidthByXScale` | 使用变换后局部 X 轴长度 `hypot(A,B)` |
| `WidthByYScale` | 使用变换后局部 Y 轴长度 `hypot(C,D)` |
| `WidthByAverageScale` | 使用两个奇异值的算术平均 |
| `WidthByGeometricScale` | 使用 `sqrt(abs(det))` |
| `WidthByMaxScale` | 使用较大的奇异值 |
| `WidthByMinScale` | 使用较小的奇异值 |

除 `WidthFollowTransform` 外，中心线先变换，再使用一个标量设备空间线宽
重新描边；虚线长度与 `ArcTolerance` 也按同一比例缩放。

## 重叠诊断

`DetectOverlaps` 使用包围盒预筛和分离轴定理，只报告正面积相交。共享边、
共享顶点不算错误重叠。

`RenderOverlap` 不是简单按覆盖次数涂红：它先用 Sutherland-Hodgman 裁剪
求出每个真实相交三角形对的交集多边形，再对该交集做抗锯齿上色。因此
内部斜接共享边不会被误报。演示图底部保留了一处故意穿越的路径，用来
验证红色高亮确实工作。

## 测试覆盖

`go test ./...` 覆盖 miter 回退、圆帽/方帽面积、零长度圆点、重复点、
虚线区间、闭合虚线接缝、锐角无重叠、非均匀缩放策略、AA 边缘、共享边
不误报以及真实重叠红点。
