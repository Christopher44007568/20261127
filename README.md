# 距离约束的变量字重扩张

纯 Go 标准库实现的粗笔画变量字重实验。简单沿法线平移顶点或使用无限斜接，
在锐角、窄字腔和窄槽处会产生超长 miter、偏移边交叉，甚至让字腔塌缩后反向
增长。本项目用欧氏线段距离做间隙估计，用内侧裁剪 / 外侧斜接限制 / bevel
修正拐角，并在写出轮廓前执行精确的线段自交与拓扑签名校验。

不依赖 FreeType、CGO、第三方几何库或图像库；PNG 使用 `image/png` 生成。

## 运行

```bash
go test ./...
go run .
go run . -weight 820 -samples 4 -out artifacts
```

参数：

- `-weight`：`100..900` 字重轴，`400` 为原轮廓，`900` 对应 30 个设计单位的扩张半径。
- `-samples`：每像素每轴的超采样数，默认 4。
- `-out`：产物目录，默认 `artifacts`。

## 生成结果

运行后会生成：

- `artifacts/original_outline.png`：原始 A 字形外轮廓、字腔和顶点。
- `artifacts/expanded_outline.png`：`wght=900` 的距离扩张结果；橙色线为原轮廓。
- `artifacts/self_intersection_highlight.png`：
  - 红色无限斜接“朴素膨胀”在极限距离下的轮廓；
  - 黄心红圈标出真实非相邻边交叉 / 接触点；
  - 绿色为拓扑安全极限修正结果；
  - 蓝色为当前字重的修正结果。
- `artifacts/font_size_continuity.csv`：`12..128px` 的字腔向量面积。
- `artifacts/font_size_area.svg`：字号—字腔面积连续曲线。
- `artifacts/weight_metrics.json`：扩张距离、拐角动作、拓扑报告和全部采样。

## 几何流程

1. `Outline.Canonical` 将外轮廓规范为 CCW、字腔规范为 CW。此时所有轮廓的
   `RightNormal` 都指向填充材料之外，也就是字重增加时的扩张方向。
2. `PointSegmentDistance`、`DistanceToOutline` 和
   `MinimumNonAdjacentDistance` 提供无符号距离场与保守碰撞半径。
3. `OffsetContour` 为每个拐角求相邻偏移线交点：
   - 内侧偏移交点用于裁剪，避免两条偏移边交叉；
   - 外侧 miter 超过 `MiterLimit` 时退化为 bevel；
   - 交点若越过相邻边长度，标记为本地不安全，防止锐角字腔整体塌穿；
   - 也提供 `JoinRound` 圆弧拐角。
4. `FindOutlineIntersections` 检测 proper cross、端点 touch 和共线重叠，
   相邻共享顶点的边不会误报。
5. `ExpandOutline` 先尝试请求距离；若出现交叉、方向反转、字腔面积失效或
   嵌套签名变化，则在 `[0, requested]` 中二分最大安全距离。演示的 `900`
   字重请求距离为 30，无需回退；`47.5` 的压力测试会回退到约 43.6。

## 字号连续性

字形使用 `UnitsPerEm = 1000`。字号为 `S` 时缩放比例为：

```text
scale = S / 1000
counter_area_px = counter_area_units * scale²
```

面积由 `Contour.SignedArea` 的鞋带公式在向量空间计算，而不是统计整数像素，
因此不会受栅格化取整影响。`font_size_continuity.csv` 中物理像素面积随字号
二次连续变化，归一化字腔面积保持不变。测试会对 `12..128` 之间每 `0.5px`
采样并校验该二次关系、正面积和拓扑有效性。

## 测试覆盖

`go test ./...` 覆盖：

- 点到线段距离和投影区间；
- 线段交叉、共线分离和 bowtie 自交定位；
- 字形方向、字腔方向和初始无交叉；
- `400..900` 字重下轮廓数、嵌套签名、方向和无自交；
- 字腔面积随字重单调减小且无跳变；
- 朴素极限 miter 必然产生高亮交叉，而修正器无交叉并回退到安全距离；
- 字号面积的连续二次缩放；
- PNG/CSV/JSON/SVG 产物可写出且非空。

## 范围说明

为保持零依赖，输入使用已经展平的闭合多边形轮廓；贝塞尔曲线可先按弦高容差
展平为 `Contour`。拓扑校验假设源轮廓自身是简单轮廓。对于请求距离已经超过
局部特征半径的变量字重，API 的语义是“返回最大拓扑安全字重并在
`Reduced/Delta` 中报告截断”，而不是放任字腔闭合或路径自交。
