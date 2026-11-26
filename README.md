# Velvet Sheen Renderer

一个只用 Go 标准库的 CPU 天鹅绒样品渲染器。材质不是靠提高漫反射或整体曝光做逆光，
而是分别计算：

- 视角/光线掠射加权的 microfiber sheen（`pow(1 - NdotH, 5)` 的软散射旁瓣）。
- 以纤维切向 `T` 和横向 `B` 构造的各向异性 GGX 高光；旋转纤维只改变 `T/B` 基底。
- 带色调的 Fresnel 掠射项和有限的边缘纤维衰减。
- 背光透射项：`thin = edge² + 0.30·viewGrazing + 0.08·foldGrazing`，最终 radiance
  被硬限制在 `0.70`，避免整块布料靠提高曝光发白。

## 运行

```sh
go run . -out renders
go test ./...
```

默认输出 `960×800`、2×2 超采样的 P6 PPM：

- `renders/front.ppm`：正面主光，深酒红本体与折叠处 sheen。
- `renders/backlit.ppm`：后方主光，本体保持暗，只有透射边缘被抬亮。
- `renders/rotation.ppm`：与正面完全相同的灯光/曝光，纤维方向旋转约 `47°`。
- `renders/metrics.json`：覆盖率、线性亮度、边缘/内部亮度比和峰值诊断。

可调整参数：

```sh
go run . -width 480 -height 400 -aa 1 -exposure 3.15 -out preview
```

## 数值检查

当前最终样本的诊断结果：

- 正面边缘/内部亮度比 `0.686×`，边缘保持暗。
- 逆光边缘/内部亮度比 `10.445×`，但逆光整体/正面整体能量比仅 `1.766×`。
- 纤维旋转前后整体平均亮度比 `0.891×`，没有全局泛白。
- 三张图的 `over_one_fraction` 均为 `0`；逆光峰值严格不超过 `0.70`。

`go test` 会在低分辨率下重复这些约束，防止后续调整材质时退回“整体提亮”的做法。
