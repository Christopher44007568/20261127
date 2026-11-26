# Glyph SDF / MSDF Lab

用纯 Go 标准库把 TrueType 字形和几何测试字形转换成单通道 SDF、三通道 MSDF，
并输出覆盖率重建、小字号样例、多缩放级别拼图和尖角断裂检测结果。

## 运行

```sh
go run ./cmd/glyphsdf
go test ./...
```

不需要 `go get`，也不依赖 ImageMagick、msdfgen、FreeType 或 CGo。
程序会尝试读取系统已有的 DejaVu Sans Bold；找不到 TTF 时使用内置 `A`
和几何样本，因此核心 PNG 产物仍可生成。

## 产物

运行后写入 `artifacts/`：

- `sdf-field.png`、`msdf-field.png`：两种距离场纹理。
- `coverage-sdf.png`、`coverage-msdf.png`：高倍覆盖率重建与红色断裂标记。
- `sdf-msdf-montage.png`：总拼图，包含 12/15/20/28px 小字、真实 `A`/`R`、
  重叠双三角和密集尖角梳。
- `zoom-levels.png`：同一个 `A` 在 3×、6×、12×、24× 下的 SDF/MSDF 对比。
- `*-sdf-field.png`、`*-msdf-field.png`、`*-sdf-break.png`、`*-msdf-sharp.png`：
  每个样本的独立场图和重建图。
- `metrics.json`：每个尖角的角度、SDF/MSDF 尖端损失像素和红色异常像素数。

## 实现要点

- `internal/fontx`：最小 TTF/TrueType 解析器，支持 cmap format 4/12、
  `loca`、二次曲线、增量坐标、基础复合字形和 `hmtx` 字宽。
- `internal/geom`：向量、贝塞尔展平、非零环绕数、线段求交，以及重叠轮廓的
  相交分割和内部接缝剔除。
- `internal/field`：真实欧氏 SDF；次通道边掩码 MSDF；尖角处按凸/凹关系合并
  两条邻边的伪距离；双线性采样后取 `median(R,G,B)`。
- `CorrectSigns`：离线 MSDF error correction。它在每个纹素格内密集对照精确
  矢量填充，把中值符号错误修正回 RGB 通道，因此放大后不会只剩柔和圆角。
- `internal/render`：子采样覆盖率、抗锯齿、红色缺失区域叠加、尖角探针和
  5×7 位图标签。
- `internal/atlas`：统一基线的小字图集与排版渲染。

判定上，沿尖角内角平分线从尖端向外探查：连续约 3 个输出像素仍无法恢复为
正距离/足够覆盖率，就记为尖角断裂并画红圈、红十字和红色缺失像素。当前样本
中 SDF 会稳定复现圆角/断裂，MSDF 的断裂计数为 0。
