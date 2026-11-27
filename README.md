# 各向异性金属渲染（纯 Go 标准库）

用 Go 标准库从零实现的各向异性 GGX 金属渲染器，零外部依赖。运行
`go run .` 会在 `artifacts/` 下生成三张参考图。

## 输出

- `artifacts/sphere_reference.png`：同一材质的两个球，纤维方向分别为
  0° 与 90°，高光随切线旋转 90°（横带 ↔ 竖带）。
- `artifacts/brushed_plate.png`：拉丝金属板，带纤维方向箭头与确定性
  拉丝微粗糙度纹理，0°/90° 两个视角对照。
- `artifacts/roughness_ladder.png`：8 档线性粗糙度（0.05–0.88）球，
  低机位掠射观察，验证能量补偿与掠射稳定性。

## 实现要点

- **可旋转切线**：`makeFrame`/`rotateTangent`（`vec.go`）绕法线旋转
  切线 T，副切线 B = N × T；材质的 `AngleRad` 决定纤维方向。
- **各向异性 GGX**：Burley 2012 NDF 与各向异性 Smith 联合遮蔽 G2
  （`brdf.go`），粗糙度/各向异性映射为切向与副切向 α。
- **能量补偿**：GGX 重要性采样预计算方向反照率 E(α, μ) 与平均反照率
  E_avg（`lut.go`），再按 Kulla-Conty / Filament 形式补回多散射能量，
  保证高粗糙度不变暗。
- **掠射稳定**：N·V、N·L 设最小夹取值，所有 BRDF/补偿分量逐项
  `max(0, …)`，杜绝负能量、NaN 与白斑；`g2Aniso`、NDF 均有下界保护。
- **灯光**：太阳圆盘 + 长条软箱（`env.go`），黄金角低差异分层采样，
  阴影射线跳过所属物体避免自相交；HDR 上做高斯后处理消除窄高光闪烁。
- **管线**：ACES 胶片色调映射 + sRGB 编码，`image/png` 输出。

## 文件

- `vec.go` 向量/切线框架，`brdf.go` BRDF，`lut.go` 能量 LUT
- `scene.go` 球/圆角板 SDF 几何，`env.go` 灯光，`camera.go` 相机
- `renderer.go` 着色与色调映射，`overlay.go` 标签/箭头，`scenes.go` 三场景
- `noise.go` 确定性拉丝噪声，`main.go` 入口
- `brdf_test.go` 单元测试（非负、NDF 归一化、切线旋转、掠射稳定等）

## 验证

```
go test ./...
go run .
```
