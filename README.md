# Organic UV Seam Lab

纯 Go、零外部依赖的有机模型自动 UV 展开实验。程序内置一个带鼻梁、眼窝、眉弓、颧骨、嘴唇、下颌、耳朵和脖颈的程序化头部，完成曲率/自遮挡分析、正面保护分缝、LSCM 风格参数化、拉伸度量以及 CPU 光栅化渲染。

```bash
go run .
go test ./...
go vet ./...
```

默认把结果写入 `artifacts/`。也可以指定目录和遮挡采样数：

```bash
go run . -out artifacts -ao-samples 24
```

## 生成结果

| 文件 | 内容 |
| --- | --- |
| `artifacts/uv_islands.png` | 打包后的 UV 岛、三角网格、岛编号和琥珀色切缝边界 |
| `artifacts/checker_model.png` | 正面、四分之三侧面、背面的透视校正棋盘格模型 |
| `artifacts/stretch_heatmap.png` | 正面/背面拉伸热图与 UV 空间失真热图 |
| `artifacts/seam_plan.png` | AO 底色上的正面、侧面、背面分缝路径 |
| `artifacts/unwrapped_head.obj` | 带 `vt` UV 坐标的展开模型 |
| `artifacts/metrics.json` | 切缝、翻面、面积比、L2/L-inf 拉伸和显式检查项 |
| `artifacts/report.txt` | 人类可读的指标摘要 |

当前演示网格约为 3.7k 顶点、7.5k 三角形。一次典型结果为 2 个 UV 岛、125 条切缝边、0 个翻面、0 条可见正面切缝；`L2 P95` 约 1.5。具体数值以重新运行后的 `artifacts/metrics.json` 为准。

![UV islands](artifacts/uv_islands.png)

![Checker model](artifacts/checker_model.png)

![Stretch heatmap](artifacts/stretch_heatmap.png)

## 需求对应

- **曲率与遮挡分缝**：`signals.go` 用余切权重 Laplacian 估计曲率，用法线夹角补充硬折痕；`bvh.go` 构建 BVH 并从每个顶点沿确定性余弦半球发射自遮挡射线。
- **正面切缝最少**：`IsProtectedFacePoint` 把面中/额头前方区域设为高代价保护区，保护边的代价为 `+Inf`。最短路径只沿后脑和侧面绕行，另一条闭合分缝放在下颌/脖颈高遮挡带。当前正面可见切缝边为 0。
- **LSCM 风格参数化**：`lscm.go` 在每个三角面建立 Cauchy–Riemann 最小二乘方程，固定相距最远的两个 UV wedge，再用 Jacobi 预条件 CG 解法方程。
- **岛不翻面**：`topology.go`/`unwrap.go` 用非切缝边并查集生成 UV wedge；展开后逐三角形检查 UV 有向面积。若出现负面积，会沿非保护边自适应加分并重新求解。当前结果和回归测试都要求 `flipped_faces = 0`。
- **拉伸评估**：由三维第一基本形式和 UV 度量计算 Jacobian 奇异值，输出面积比、`s_min/s_max`、Sander L2/L-inf stretch，以及面积失真加角度各向异性的热图分数。
- **渲染**：`render_model.go` 实现透视相机、深度缓冲、屏幕空间包围盒光栅、透视校正 UV/法线插值；棋盘格支持导数抖动抗锯齿。UV 图和热图均由标准库 `image/png` 输出。

## 代码结构

- `mesh.go`：程序化有机头部、法线、包围盒和 OBJ 写出入口。
- `topology.go`：边/面邻接、切缝岛 flood-fill、UV wedge 并查集。
- `bvh.go`：三角网格 BVH、AABB 和射线求交。
- `signals.go`：曲率、光线 AO、正面保护掩码和切缝代价。
- `seam.go`：下颌环、后脑 Dijkstra 最低代价路径。
- `lscm.go`：LSCM 方程、稀疏算子和 CG 求解。
- `unwrap.go`：展开流程、翻面修复和拉伸度量。
- `pack.go`：保持岛间相对纹素密度的 shelf atlas 打包。
- `render_uv.go`、`render_model.go`、`render_shade.go`、`render_sheets.go`：UV、棋盘格和热图渲染。
- `unwrap_test.go`：硬约束和产物有效性测试。

## 设计说明

正面不是简单“事后把接缝颜色藏起来”，而是在代价阶段禁止切缝进入脸中心。主要切口是一条从颅顶沿后脑到颈底的路径，加上一条位于下颌下方/脖颈的闭合环；它们把模型分成头颈主岛和脖颈岛。这样正面视图没有琥珀色切缝，侧面只能在下颌线以下看到环缝，后脑承担必要的展开切口。

拉伸度量在打包前进行全局等面积尺度归一化，因此面积比的全网格加权均值接近 1。打包只使用平移和统一缩放，不做逐岛非均匀拉伸；最终 PNG 中的棋盘格密度可以直接比较。
