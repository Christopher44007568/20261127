# Bloom 中间格式对照器

这是一个不依赖图形 API 的纯 Go 实现。它让同一条 Bloom 节点图分别经过
`RGBA16F`、`R11G11B10` 和 `sRGB8` 中间纹理，然后输出每种格式独立的 HDR
数据、可视化对照图、量化误差热图和回载校验用 PPM。

## 运行

```bash
go run ./cmd/bloomdiff -out out
go test ./...
```

可选参数：`-width`（默认 480）、`-height`（默认 180）、`-out`（默认 `out`）。

## 输出

```text
out/
  summary.json
  rgba16f/
    hdr_compare.pfm        # 真实线性 HDR 位图，float32 PFM
    hdr_compare.png        # 左：float64 参考；右：该格式结果，仅供显示
    quantization_error.png # 最终结果对参考的绝对误差热图
    reload.ppm             # 16-bit P6，由该格式结果单独生成
    stats.json             # 逐节点量化、截断缓存、PPM 回载统计
  r11g11b10/
    ...
  srgb8/
    ...
```

`hdr_compare.png` 的两侧在写入前都转换为同一种 SDR 显示值，因此它只是可视化，
不是中间纹理。真正的 HDR 数据在各自的 `hdr_compare.pfm` 中，每个文件只属于一种
格式。`reload.ppm` 写入后会立即重新读取，`stats.json` 中的
`ppm_reload.mismatch_count` 和 `max_code_error` 用于确认回载一致性。

## 节点图与格式协商

Bloom 链为：

```text
scene -> threshold -> blur.horizontal -> blur.vertical -> scale
   \                                                        \
    +-----------------------> composite <-------------------+
```

`Graph.Negotiate(format)` 从 sink 反向遍历 DAG，要求每个节点能输出请求格式、每条
下游边能接收该格式，并生成包含每个节点实际输出格式的 `Plan`。执行时还会拒绝任何
与计划不一致的纹理。因此一次执行中的所有中间位图和最终位图都只有一种格式。

每个节点的输出都会先编码成目标纹理，再由下游节点解码。误差热图比较的是完整量化
链的结果与相同节点图的 float64 无量化参考。

## 格式与越界策略

| 格式 | 线性目标范围 | 存储 |
| --- | --- | --- |
| `RGBA16F` | `[0, 65504]` | 每通道 binary16，RGBA 8 bytes/pixel |
| `R11G11B10` | R/G `[0, 65024]`，B `[0, 64512]` | 5-bit 指数无符号浮点，4 bytes/pixel |
| `sRGB8` | `[0, 1]` | sRGB 传递函数后 8-bit，RGB 3 bytes/pixel |

所有格式都采用“截断并缓存”策略：

- 小于下限或大于上限的通道被截断到目标范围。
- 每个节点分别累计 `low_clamp_count`、`high_clamp_count`、溢出量总和与最大溢出量。
- `NaN`/`Inf` 单独计数，不会生成不可序列化的统计。
- 量化误差按截断前的线性原值计算，所以截断损失不会伪装成精确往返。

这些统计位于各目录的 `stats.json` 和顶层 `summary.json` 中。
