# facemorph

面部形态目标（morph target / blendshape）混合器，仅使用 Go 标准库实现。

## 解决的问题

叠加多个表情目标时只混合位置会让笑纹（鼻唇沟）处保留基础脸法线，导致着色方向
错误。本实现对每个顶点同时混合**位置、法线、切线**三组数据，并对混合后的法线
重新归一化、对切线相对新法线重新正交化，保持切空间正交。

## 混合模型

每个目标存储相对基础脸的逐顶点 delta（`PosD` / `NormalD` / `TanD`），最终姿态为

```
pose = base + Σ effectiveWeight[i] * delta[i]
```

权重处理分两步，见 `blender.go`：

1. **钳制**：NaN 按 0 处理，其余权重钳制到目标的 `[Min, Max]`，极端表情不会
   把面部折叠翻转。
2. **归一化**：钳制后权重之和大于 1 时整体除以该和；单目标或总和不超过 1 时
   保持原权重，因此单目标姿态完全不变。

所有目标权重为 0 时直接返回基础脸的逐字节拷贝，位置、法线、切线均不经过浮点
往返，保证逐顶点回到基础脸（`TestAllOffReturnsBasePerVertex`）。

## 渲染产物

`go run .` 在 `artifacts/render/` 输出六个场景：单目标（微笑、张嘴）、组合目标
（三目标、全脸四目标）、极端权重钳制、全关恒等。每个场景包含：

- `<场景>.ply`：带混合法线的 ASCII PLY 网格
- `<场景>_normal_diff_rgb.png`：法线差 RGB 图，差值 `(dx,dy,dz)` 映射到 0..255，
  全关时为均匀中灰 `(128,128,128)`
- `<场景>_normal_diff_mag.png`：法线夹角热力图（45° 饱和）
- `report.json`：原始/钳制/有效权重与产物路径

## 使用

```bash
go test ./...          # 7 个测试：恒等、单目标、归一化、钳制、NaN、正交框、笑纹回归
go run .               # 默认输出 artifacts/render，512px 差图
go run . -out out -size 1024
```

## 文件结构

- `vec3.go`：三维向量
- `mesh.go`：网格与形态目标 delta 类型
- `blender.go`：钳制、归一化与位置/法线/切线混合
- `builder.go`：程序化基础脸（网格、UV、微分得到切线与法线）
- `morphs.go`：微笑、鼻唇沟、抬眉、张嘴四个目标
- `export_ply.go` / `normalmap.go`：PLY 与 PNG 输出
- `main.go`：场景编排与 JSON 报告
