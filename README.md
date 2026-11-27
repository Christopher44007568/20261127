# 足迹粗糙度升采样与能量补偿

一个仅使用 Go 标准库的确定性软件渲染演示：同一张程序化粗糙度贴图映射到三个金属球，
从左到右分别是：

1. **原值采样**：最近邻读取原始粗糙度，缩小后高频粗糙度造成随机窄高光，即点状闪点。
2. **错误过滤**：只对粗糙度的算术平均值建 mip，再使用 `alpha = mean(r)^2`。
   这会忽略足迹内方差，结果偏光滑，高光仍可能碎裂。
3. **足迹修正**：由屏幕空间 UV 微分构造纹理足迹，连续采样一阶/二阶矩，
   使用 `alpha = E[r²] = E[r]² + Var(r)`，并加入球面曲率方差与多次散射能量补偿。

## 运行

```sh
go run .
go test ./...
```

默认写入 `artifacts/`：

- `near.png`：近距离三球对比，`1500×640`。
- `far.png`：远景三球对比，`1500×640`。
- `preview.png`：中间视距预览。
- `continuity.png`：修正模式的连续视距扫描。
- `roughfoot-report.json`：闪点、高频误差、有效粗糙度和亮度连续性指标。

可覆盖参数：

```sh
go run . -out renders -tex-size 512 -lut-samples 64
```

`go.mod` 没有任何第三方依赖，也不需要图片、字体或模型资源。

## 足迹与粗糙度矩

`RoughnessTexture` 的每一级 mip 保存：

- `MeanR = E[r]`
- `MeanA = E[r²]`，即 GGX 的 `alpha`
- `MeanA2 = E[r⁴]`

像素着色时，用相邻像素的球面 UV 微分得到
`du/dx、dv/dx、du/dy、dv/dy`，进而计算足迹纹素长度、面积和连续 LOD。
两级 mip 之间做三线性统计插值，因此 mip 边界不会产生亮度突变。

错误过滤与修正的核心差别是：

```text
错误：alpha_wrong   = E[r]²
正确：alpha_foot    = E[r²] = E[r]² + Var(r)
```

细微的高低粗糙度混合区域因此会提高有效粗糙度，把孤立窄高光展开成稳定的宽高光，
而不是保留点状火花。球面法线微分得到的曲率方差也在 `alpha` 空间相加，
用于抑制几何边缘缩小后的高光闪烁。

## 能量补偿

`BuildEnergyLUT` 使用确定性 Hammersley 样本和 GGX VNDF 重要性采样，
预计算白炉方向反照率 `E(mu, roughness)`。修正模式使用 Kulla–Conty 风格的
多次散射项：

```text
f_ms = (1-E_o)(1-E_i) F_avg² E_avg
       / (π (1-E_avg) (1-F_avg(1-E_avg)))
```

它补偿单散射 Cook–Torrance GGX 在高粗糙度和掠射角损失的能量，
避免粗糙度升采样后金属球变暗或近远亮度跳变。

## 自检指标

一次确定性运行的报告通常显示：

- 原值模式有数百个可检测闪点，高频误差约为修正模式的 10 倍。
- 足迹修正模式的闪点检测数为 `0`。
- 修正模式的平均有效粗糙度高于朴素 mip 均值。
- 连续视距扫描中，相邻半径的平均亮度最大相对跳变低于 `1%`
  （测试预算为 `5%`）。

## 代码结构

- `roughness.go`：程序化粗糙度贴图、mip 矩链、UV 足迹与粗糙度解析。
- `noise.go`：确定性哈希、值噪声、FBM 和细胞斑点。
- `brdf.go`：GGX、Smith、Schlick、VNDF 采样和能量补偿 LUT。
- `scene.go` / `shade.go`：三球场景、UV 微分、棚拍灯光和材质。
- `frame.go`：HDR 帧、ACES、sRGB 输出和 bloom。
- `metrics.go` / `sweep.go`：闪点与连续性量化、视距扫描。
- `roughness_test.go` / `render_metrics_test.go`：矩统计、LOD 连续性和渲染回归。
