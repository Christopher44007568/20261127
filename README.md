# 自适应三次贝塞尔描边

本项目用 Go 标准库实现了一条不依赖固定细分步数的三次贝塞尔描边管线：

1. 将曲线变换到最终像素空间，按**解析弦高**自适应展平；
2. 用常量宽度四边形、自适应圆角连接与端帽生成可填充的描边网格；
3. 用 **16×16 子像素覆盖率**做 nonzero 扫描线归一化，合成抗锯齿 PNG。

当放大或曲线变长时，细分数量随实际弦高自动增加。曲线端点始终取精确的
`B(0)` 和 `B(1)`，不会因为固定步长而在末端过冲、变粗或断开。

## 图像结果

近直线场景的三张放大诊断图：

| 原曲线 | 描边网格边 | 最终覆盖率 PNG |
| --- | --- | --- |
| ![原曲线](artifacts/original.png) | ![网格边](artifacts/mesh_edges.png) | ![最终 PNG](artifacts/final.png) |

另外还有两组连续性压力测试，每组的 `original`、`mesh`、`final` 文件都在
`artifacts/` 下：

- `sharp_bend_*`：曲线中点导数为零、切线反向的尖点（约 180° 回转）；
- `long_curve_*`：展开长度约 1548 px 的长曲线，最终描边仅 8 px 宽。

所有 PNG 都是实际生成的 1600×820 图像；`artifacts/metrics.json` 记录了
每个场景的顶点数、实际最大弦高和固定 8 段的对照误差。

## 运行

需要 Go 1.23 或更新版本，不安装任何第三方包：

```bash
go test ./...
go run ./cmd/bezier-demo -out artifacts -samples 16
```

`-samples N` 控制每个像素每个轴的覆盖率采样数，默认 16。示例只使用
`image/png`、`image/color`、`encoding/json` 等标准库。

## 实现要点

### 1. 解析弦高而不是估采样

对参数区间 `[t0, t1]`，令 `d = P3 - P0`、`q(t) = cross(B(t) - P0, d)`。
曲线到弦线的最大垂距就是 `max |q(t)| / |d|`。由于 `q'(t)` 是二次多项式，
通过求根即可得到所有内部极值，再与端点比较。实现见
[`ChordHeight`](flatten.go)。

仅看垂距会把沿同一直线折返的曲线误判为“平”。因此每个候选区间还会检查
控制点在弦方向上的投影是否单调；不单调就继续二分，避免零弦高折返和尖点
被塌缩。

### 2. 按像素容差递归展平

[`AdaptiveFlattenDetailed`](flatten.go) 在屏幕坐标中用 de Casteljau 精确
二分，直到区间满足容差。结果携带原始参数 `T`，输出顺序稳定且首尾精确。
最大深度 48 只作为极端退化输入的防爆栈保护，不是固定细分步数。

三个演示场景的实际结果：

| 场景 | 容差 | 自适应顶点 | 最大实际误差 | 固定 8 段误差 |
| --- | ---: | ---: | ---: | ---: |
| 近直线 | 0.08 px | 33 | 0.023 px | 0.375 px |
| 尖点 | 0.25 px | 31 | 0.247 px | 3.829 px |
| 长曲线 | 0.10 px | 57 | 0.093 px | 5.364 px |

### 3. 描边顶点与连接

[`StrokePolyline`](stroke.go) 为每个展平边生成左右偏移后的常量宽度四边形；
在内部顶点生成外侧圆角扇形，弧的弦高同样按容差自适应计算。端帽使用半径为
半线宽的圆形端帽，所以末端不会超出给定线宽。

所有网格多边形统一为正向绕序，并按 nonzero 规则取并集。尖点处相邻四边形
与连接扇形相互重叠，不会漏出孔洞；重叠区域也只记一次覆盖率。

### 4. 覆盖率渲染

[`Coverage`](raster.go) 在像素内建立 `samples × samples` 采样网格：每个
子扫描线求多边形交点并排序，按 nonzero winding 填充覆盖计数。最后把
`coverage / samples²` 与背景色混合。这样得到的是面积覆盖率，而不是简单
的“像素是否在三角形内”二值栅格化。

## 文件结构

- `geometry.go`：点、三次贝塞尔求值/导数/参数区间和屏幕变换；
- `flatten.go`：解析弦高与自适应展平；
- `stroke.go`：描边四边形、圆角连接、端帽和网格多边形；
- `raster.go`：覆盖率采样、扫描线填充和 PNG 输出；
- `cmd/bezier-demo/`：三个场景的生成入口、HUD 标注和指标 JSON；
- `*_test.go`：弦高、展平、尖点连续性、描边宽度与覆盖率测试。

## 验证

```bash
$ go test ./...
ok  example.com/bezierstroke

$ go run ./cmd/bezier-demo -out artifacts -samples 16
near_line: 33 adaptive vertices, max error 0.0234 px
sharp_bend: 31 adaptive vertices, max error 0.2472 px
long_curve: 57 adaptive vertices, max error 0.0932 px
```
