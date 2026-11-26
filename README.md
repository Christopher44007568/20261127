# renderlifetime

一个仅使用 Go 标准库的帧资源生命周期演示。程序用引用计数、延迟释放队列和
帧边界复现三种结果：

- 正常回收：消费 pass 持有租约，纹理在边界后回收，下一帧与上一帧无关。
- 提前释放：缺少消费 pass 的引用，纹理在同一帧内被销毁，后续 pass 输出黑块。
- 延迟回收：旧纹理多存活一帧且未清空便被复用，下一帧残留上一帧内容。

错误或失效的纹理区域用洋红色斜纹和角标高亮。

## 运行

```sh
go run .
go test ./...
```

默认输出到 `artifacts/`，也可使用 `go run . -out /some/dir`。

## 输出

| 文件 | 内容 |
| --- | --- |
| `artifacts/normal.ppm` | 正常路径：帧 0、帧 1、独立新生帧 1 |
| `artifacts/early_release.ppm` | 提前释放：黑块与洋红色失效标记 |
| `artifacts/delayed_reclaim.ppm` | 延迟回收：帧 0 残影与洋红色陈旧标记 |
| `artifacts/lifetime_report.txt` | 引用、回收、队列和像素差异统计 |

每张 PPM 是 1312×516 的三联图。左侧为帧 0，中间为复用资源后的帧 1，右侧为
全新管理器直接渲染的帧 1。正常图中中间与右侧内容逐像素一致；失败场景中
中间图与右侧参考图的差异会被洋红色标出。

## 生命周期

资源状态为 `live -> retired -> reclaimed`：

1. 每个 producer/consumer pass 显式 `Acquire`，得到独立 `Lease`。
2. 每次 `Release` 只减少一个引用；最后一个引用释放后进入延迟队列。
3. `BeginFrame` 先处理上一帧边界，再允许本帧 pass 运行。
4. 正常策略在下一个帧边界回收；迟延策略会多保留一帧用于诊断。
5. 回收会使资源 ID 与 generation 失效，旧租约不能再次采样。

`early_release` 诊断故意省略 consumer 租约并立即释放，`delayed_reclaim`
诊断故意复用未清空的 retired surface。这两种错误路径与正常路径共享同一套
渲染代码，便于直接比较。

## 代码结构

- `lifecycle.go`：引用计数、延迟队列、帧边界和 generation。
- `render.go`：producer/consumer pass 顺序及失败注入。
- `scene.go`：透明信号层、合成背景和参考帧。
- `sheet.go`：三联图布局、统计徽标和洋红色诊断标记。
- `surface.go`：RGBA 光栅、alpha 合成和 P6 PPM 编码。
- `font.go`：内置 5×7 点阵字体，不依赖外部字体。
