# Post-processing graph culling in Go

纯 Go、仅标准库的 CPU 后处理节点图示例。图包含 base、bloom、体积雾、UI 与
preview 节点。裁剪只删除不参与请求输出的 pass，不改变最终图像。

## 图结构

```text
scene -> hdr ---------------------------> bloom-threshold -> blur-h -> blur-v -> bloom
     -> depth -> fog-density -> fog-scatter -> fog
                         bloom .......... weak order .........> ui-layer -> ui
hdr + bloom + fog-density + fog + ui -> beauty -> tonemap -> final
bloom / fog / ui -> 三个独立 preview root
```

`bloom`、`volumetric-fog`、`ui` 是三条可选路径。三条 preview 输出不是 `final`
的依赖，因此裁剪时会被删除；三条效果路径本身仍由 `final` 的强依赖保留。

## 裁剪语义

- `ExecuteReachable`：从请求的 root 沿强输入反向遍历，只保留实际贡献像素的
  pass，并生成拓扑序。
- `ExecuteAllActive`：执行所有可用 pass，作为未优化参考。未请求的输出会在该
  pass 执行后立即回收。
- `WeakAfter` 只约束执行顺序，不传播存活状态。弱依赖目标被裁掉时，不会连带
  删除仍被其他 live pass 强读取的生产者。
- `DisabledOutputs` 中的返回不会成为 root，也绝不会申请纹理；多输出 pass 在
  可达模式下只申请被读取或请求的输出。
- 内部纹理带引用计数，在最后一个读取者完成后归还纹理池。最终 root 保留到
  `Frame` 返回为止。

每次执行都会在 `ExecutionReport` 中记录实际执行的节点、被裁节点及原因、纹理
申请/释放、峰值占用和结束时残留的中间纹理。

## 运行

```bash
go test ./...
go run ./cmd/postfx-demo -width 480 -height 270 -out artifacts/render
```

命令会输出实际执行节点，并生成：

- `full.png`：`ExecuteAllActive` 的全量执行结果。
- `culled.png`：`ExecuteReachable` 的裁剪执行结果。
- `difference.png`：两图逐通道绝对差。正常结果全黑，最大通道差为 `0`。
- `bloom.png`、`fog.png`、`ui.png`：三条可选路径的独立预览。
- `report.json`：节点级执行清单、裁因、纹理生命周期和四项自动化检查。

示例运行中全量执行 12 个节点，裁剪后执行 9 个节点，三张 preview 节点被删除；
禁用返回 `fog-debug`、`ui-debug` 未分配纹理，所有中间纹理均在结束时释放。
