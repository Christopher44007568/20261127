# 骨骼动画低码率压缩（纯 Go 标准库）

针对“脚部接触与快速转身最先失真”的问题，本项目把位置、旋转、时间
三个通道分别量化，并以**关键姿态完全无损**为硬约束，而不是只优化平均
角度误差。

## 运行

```bash
go run .
go run . -pos-bits 6 -tolerance 0.015 -max-gap 10
go test ./...
```

所有产物写入 `artifacts/`：

- `original_animation.json` / `compressed_animation.json`：局部姿态数据
- `compressed.sakc`：自定义二进制压缩载荷
- `joint_error_heatmap.png`：帧 × 关节的世界空间位置误差热图（毫米）
- `foot_slip_curve.png`：接触期脚底滑移曲线（绿色原始恒为零）
- `overlay.gif`：侧视 + 俯视下原始（黑）与压缩（红）骨架叠加
- `stats.txt`：码率、压缩比、逐关节误差、滑移与关键位校验结果

## 三个独立量化通道

- **时间量化**（`timequant.go`）：在关键姿态硬锚点之间做贪心抽帧，
  位置用米、旋转用四元数弦误差共同判定插值区间；接触帧的踝关节误差
  施加 6 倍权重，使脚底落地区域保留更密的采样。
- **位置量化**（`quantize.go` / `codec.go`）：每个关节、每个分量独立
  min–max 包络，标量定点编码，位数由 `-pos-bits` 控制。
- **旋转量化**（`quantize.go`）：smallest-three 四元数打包，最大分量
  代数重建且恒取非负，省去符号位；每分量 11 bit。

## 关键姿态为何能完全一致

关键帧（片段首尾、转身起止、每次抬脚/落脚）跳过全部量化器，按原始
`float64` 字节写入独立的 key block。解码时逐字节还原，因此
`VerifyKeyPoses` 用 `!=` 直接比较即可保证位级一致，测试覆盖 4/8/12 bit。

## 脚底滑移度量

合成片段在接触期把脚踝世界坐标锁定到固定锚点，所以原始滑移为浮点零。
压缩后在每个接触区间内，以区间起点的解码脚踝为锚点累计漂移
（`metrics.go` 的 `ComputeSlip`）。接触状态本身按区间边界编码
（`contacts.go`），不会因抽帧而把摆动期误判成接触。

## 文件组织

| 文件 | 职责 |
| --- | --- |
| `math.go` | 向量、四元数、nlerp |
| `skeleton.go` | 18 关节层级与前向运动学 |
| `footstep.go` / `synth.go` | 行走-转身-行走合成、两骨 IK |
| `bitio.go` | MSB 比特读写 |
| `quantize.go` | 位置标量与四元数 smallest-three |
| `timequant.go` | 接触感知贪心抽帧 |
| `codec.go` / `decode.go` | SAKC 二进制编解码与全帧重建 |
| `contacts.go` | 接触区间提取与逐帧重建 |
| `metrics.go` | 关节误差热图数据、滑移、关键位校验 |
| `raster.go` / `font.go` / `render_reports.go` / `render_gif.go` | 纯标准库位图与 GIF 渲染 |
| `clipio.go` | JSON 导出 |

仅依赖 Go 标准库（`encoding/binary`、`image`、`image/gif`、`image/png`
等），无任何第三方包。
