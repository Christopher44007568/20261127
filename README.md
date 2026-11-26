# Geodesic Curvature Partition

纯 Go 标准库实现的网格测地线分区工具。程序从种子点出发计算加权测地线距离场，用多种子优先队列完成分水岭/Voronoi 归属，再通过曲率方向感知的能量最小化松弛边界，最后输出种子分布、区域颜色、边界线和距离热图。

## 运行

```bash
go run .
go test ./...
```

默认生成 `outputs/` 与 `outputs/changed_seeds/` 两组结果，用不同种子验证连通性：

- `seed_distribution.png`：地形晕渲、种子位置和编号。
- `region_colors.png`：每个连通区域的颜色。
- `boundary_lines.png`：高曲率/凹陷背景上的区域边界。
- `distance_heatmap.png`：多种子测地线距离热图。
- `boundary_lines.svg`：矢量边界线。
- `segmentation.json`：种子、参数、翻转数、修复数和连通性报告。

自定义种子：

```bash
go run . -seeds '20,20;160,25;35,120;150,115;96,72' -out outputs/custom
```

常用参数：

```bash
go run . -width 192 -height 144 -seeds-count 8 \
  -relax-iterations 30 -feature-weight 1.05 \
  -smooth-weight 0.46 -feature-discount 0.16
```

## 算法

1. **地形与曲率**：合成包含蜿蜒凹陷、环形高曲率脊和内部孔洞的高度场；计算 Hessian、平均曲率、曲率幅值、凹陷得分和特征线切线。
2. **测地线距离场**：`container/heap` 实现 Dijkstra；边权包含水平步长和高程差。八邻域热图禁止对角穿角，四邻域分水岭保证归属链严格四连通。
3. **种子分水岭**：多种子优先洪泛等价于加权测地线 Voronoi/分水岭，种子可由测地线最远点采样自动生成，也可由 `-seeds` 指定。
4. **边界松弛**：迭代评估 `测地线归属代价 + 边界周长代价 - 方向一致的曲率/凹陷奖励`。翻转前检查旧标签移除后是否仍连通、新标签是否有相邻支撑；随后再做一轮纯周长平滑。
5. **连通性兜底**：若关闭拓扑检查或外部调用直接修改标签，`EnsureConnected` 会建立“同标签四连通分量图”，从各种子分量进行多源 BFS，把孤立分量沿相邻父分量重归属。

孔洞和对角接触不会造成伪连通：所有区域校验、分量图和兜底重归属都只使用上下左右四邻接；对角步还要求两个正交侧格均可通行。离散可行区域缺少种子时会自动补种子，严格模式可通过 `SegmentOptions.StrictSeeds` 关闭该行为。

## 主要文件

- `grid.go`：网格、孔洞和邻接规则。
- `distance.go`：加权 Dijkstra 距离场。
- `seeds.go`：最远点种子、校验和离散区域覆盖。
- `curvature.go`：曲率、凹陷和特征方向。
- `relax.go`：能量与边界松弛。
- `connectivity.go`：四连通校验、拓扑检查和分量重归属。
- `segment.go`：完整管线。
- `render*.go`、`font.go`、`output.go`：PNG/SVG/JSON 输出。
