# 渐变 / 蒙版裁剪栈

只用 Go 标准库实现的透明合成示例：同一组圆形和几何文字上依次叠加线性渐变、径向渐变与 Alpha 蒙版，输出三种混合模式、逐层图、混合顺序对比、最终 PNG 和色带/接缝检测图。

## 运行

```bash
go test ./...
go run ./cmd/gradientstack -out out
```

输出目录默认为 `out/`，不会下载或安装任何额外组件。

## 产物

- `out/final.png`：正常混合的最终 PNG（1600×1000）。
- `out/final_normal.png`、`out/final_multiply.png`、`out/final_screen.png`：三种可分离混合模式。
- `out/layers/`：背景、组裁剪蒙版、线性渐变、径向渐变、Alpha 蒙版、蒙版前栈和最终层的全分辨率 PNG。
- `out/layer_sheet.png`：逐层总览。
- `out/blend_modes.png`：三种混合模式并排对比。
- `out/order_compare.png`：`linear → radial → mask` 与 `linear → mask → radial` 的透明边缘对比，附 2× 边缘裁切。
- `out/banding_detection.png`：原始 8-bit 平台、抖动结果、检测掩码和 3×2 周期平铺接缝检查。
- `out/seamless_tile.png`：可重复的周期渐变瓦片。
- `out/report.json`：文件清单与自动检查指标。

## 合成实现

内部缓冲是 `float32` 线性光、预乘 Alpha RGBA：

```text
out.rgb = src.rgb + dst.rgb * (1 - src.a)
out.a   = src.a   + dst.a   * (1 - src.a)
```

`src.rgb` 已乘过 Alpha，因此软边不会把透明像素中的黑色带进暗背景。抗锯齿圆、描边字形和 Alpha 蒙版都只缩放预乘后的 RGBA，不会单独动 RGB。

混合模式使用 W3C 可分离混合公式的预乘形式：

```text
Co = (1 - ab) * as * Cs
   + as * ab * B(Cs, Cb)
   + (1 - as) * ab * Cb
ao = as + ab - as * ab
```

其中 `B` 分别为 `Cs`（normal）、`Cs*Cb`（multiply）和 `Cs+Cb-Cs*Cb`（screen）。由于公式始终用 `as`/`ab` 加权，半透明边缘不会因混合模式失配出现灰边。

裁剪栈保存每层覆盖率并逐层求交。场景先压入画布矩形，再压入“三个圆 ∪ ALPHA 文字”的覆盖率蒙版；渐变图层只在交集内写 Alpha。Alpha 蒙版则在栈末尾以 destination-in 方式同时缩放 RGB 和 Alpha。

## 色带与接缝

PNG 为 8-bit，编码前在 sRGB 量化域应用确定性 `Bayer 8×8 + interleaved gradient noise` 抖动，扰动幅度为 ±2 LSB。检测器把连续相同颜色的横向平台按 8 像素为阈值标记出来：

- 未抖动原始渐变：29 个平台/行。
- 抖动后：0 个平台。
- 周期瓦片在 u/v 两端的浮点亮度差：0。
- 8-bit 平铺接缝残差：1 LSB，属于量化噪声，不是数值跳档。

周期瓦片由 `sin/cos(2πu, 2πv)` 构造，左右和上下边界取值相同；因此 3×2 平铺不会在接缝处改变色阶。
