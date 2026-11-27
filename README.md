# 20261127

Repository initialized for 20261127.

## 材质 LOD 光照实验

`material/` 实现金属、布、塑料共用的 `Material` 参数集，以及低 / 中 / 高三档光照模型：

- 低档使用 `LowUniforms`，只保留主色、金属度、名义粗糙度和粗糙区间。
- 中档使用 GGX、Smith、Schlick 和直接光。
- 高档增加环境反射近似、Sheen 与 Clearcoat。
- 连续 LOD 使用 smoothstep crossfade，在 `0.5` 与 `1.5` 两个硬切换边界附近混合。

运行测试并生成全部 PNG、CSV、JSON 和 Markdown 报告：

```bash
go test ./...
go run ./cmd/material-lod -out artifacts/material-lod
```

仅使用 Go 标准库，`go.mod` 中没有外部依赖。生成结果包括三档材质球、0°–90° 视角扫掠、亮度差热力图、连续边界误差和主色 / 粗糙区间保持性报告。
