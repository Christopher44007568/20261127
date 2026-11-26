# Reversible Constrained QEM Mesh Simplification

纯 Go 标准库实现的可逆三角网格简化器。核心是 Garland–Heckbert 二次误差度量
（QEM），并加入边界边、材质接缝、锐边锁定、体积约束、翻面拒绝和逐级轮廓偏差
评估。不依赖任何第三方 Go 包。

## 运行

```bash
go test ./...
go run ./cmd/meshsimplify
```

不带 `--input` 时会生成两个可复现演示：

- `NewDetailOrb(4)`：5120 面，带高频法线起伏、体积起伏和纬度材质接缝。
- `NewWavyPatch(33,33)`：2178 面，带开放边界和中央材质接缝。

处理自己的 OBJ：

```bash
go run ./cmd/meshsimplify \
  --input path/to/model.obj \
  --out output/model \
  --ratios 1,0.75,0.5,0.35,0.22 \
  --max-volume-loss 0.005
```

OBJ 读取支持 `v`、`f`、负索引、多边形扇形三角化和 `usemtl`。UV/法线索引会被
忽略；材质编号用于识别材质接缝。

## 每级产物

以 `output/detail_orb` 为例：

- `ppm/level-XX_turntable.ppm`：4 视角转盘着色图，固定原始模型取景框。
- `silhouette/level-XX_deviation.ppm`：8 视角轮廓叠加，4×2 排布。
- `highlights/level-XX_collapsed_edges.ppm`：本级新折叠边的深度测试高亮。
- `obj/level-XX.obj`：该级紧凑 OBJ。
- `obj/restored-after-undo.obj`：执行全部 vertex split 后恢复的 OBJ。
- `silhouette-metrics.csv`：每级、每旋转视角的轮廓指标。
- `report.json`：汇总指标、保护边计数和约束校验结果。
- `collapse-history.json`：折叠记录，包含恢复拓扑所需的 stable ID。

PPM 是二进制 P6 格式。可用 ImageMagick 查看或转换：

```bash
convert output/detail_orb/ppm/level-02_turntable.ppm turntable.png
```

轮廓叠加颜色：

- 灰：原始和简化轮廓重合。
- 蓝：仅原始轮廓存在，即简化后丢失/内缩区域。
- 红：仅简化轮廓存在，即外扩区域。

## 约束与安全折叠

每条候选边在真正提交前都会模拟，并满足：

- `boundary`：只有一个邻接面的边，默认锁定。
- `material_seam`：两侧面材质不同的边，默认锁定。
- `non_manifold`：邻接面超过 2 个的边，始终锁定。
- `sharp_normal`：二面角超过 `--sharp-angle` 的法线/锐边，默认锁定。
- `LockProtectedVertices=true`：保护边两端不能作为被删除顶点，因此原保护边的
  stable endpoint 不会通过相邻折叠被间接合并。
- `FreezeLockedPositions=true`：锁定顶点位置不变，边界和材质接缝几何不漂移。
- 替换后不得出现重复三角形、退化三角形或法线反向；同时检查相对折叠前法线和
  初始法线的朝向。
- 体积项使用三角网格有向行列式。对候选折叠，体积变化关于新顶点位置是线性的，
  因此可构造精确的“零体积变化平面二次项”加入最优位置求解；另外还有累计体积
  下限，默认最多损失初始物理体积的 0.5%。

如果所有低成本边都不合法，简化器返回 `ErrNoCollapsibleEdge` 并安全停止，而不是
破坏锁边、制造翻面或越过体积预算。演示中球体在约 26.5% 面数处到达该安全下限。

## 可逆性

每次折叠写入一条 `CollapseRecord`：

- 被删除/保留顶点 ID 和旧位置；
- 保留顶点旧 quadric；
- 被移除面 ID；
- 每个受影响三角形的旧/新顶点三元组；
- 成本、体积变化和折叠后体积。

调用：

```go
s, _ := qem.NewSimplifier(mesh, qem.DefaultOptions())
rec, _ := s.CollapseOne()
// ...
_, _ = s.Undo()    // 撤销最近一次
_ = s.UndoAll()    // 精确恢复初始网格
```

测试 `TestOrbCollapsePreservesLocksVolumeAndUndo` 和
`TestBoundaryPatchLocksBoundaryEdges` 会在大量折叠后执行 `UndoAll`，并逐顶点、
逐三角形比较恢复结果。

## 轮廓指标

所有级别复用原始模型的中心和最大边长，不会因重新取景而掩盖轮廓缩水。每个 yaw
视角渲染二值遮罩，并用二维 Euclidean distance transform 计算：

- `area_ratio`：简化轮廓像素数 / 原始轮廓像素数。
- `iou`：交并比。
- `chamfer_units`：对称平均轮廓距离，已换算为模型单位。
- `hausdorff_units`：最大轮廓距离。
- `min_area_ratio`：所有旋转视角中最差的面积保持率。

## 代码布局

- `mesh.go`、`obj.go`：stable ID 网格和 OBJ I/O。
- `quadric.go`：对称 4×4 quadric 与最优位置求解。
- `simplify.go`、`collapse_prepare.go`、`collapse.go`：保护边、候选模拟、折叠和撤销。
- `topology.go`、`heap.go`：局部邻接索引和过期版本化候选堆。
- `ppm.go`、`draw.go`、`render.go`：PPM、变换、z-buffer 着色和高亮。
- `silhouette.go`、`metrics.go`：遮罩、EDT 和旋转轮廓偏差。
- `pipeline.go`、`report_io.go`：逐级产物、JSON/CSV 输出。
- `demo.go`：无外部素材的程序化演示模型。
- `cmd/meshsimplify/main.go`：命令行入口。
