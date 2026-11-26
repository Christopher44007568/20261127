# 确定性渲染 Pass 调度（Go）

同一帧的四种 pass 用三种合法顺序执行，最终 PPM 逐字节一致。实现只使用
Go 标准库，不需要安装额外组件。

## 交付物

- `artifacts/schedule_ui_before_tonemap.svg` + `.ppm`
- `artifacts/schedule_tonemap_before_ui.svg` + `.ppm`
- `artifacts/schedule_ui_parallel_tonemap.svg` + `.ppm`
- `artifacts/schedules.json`：边、顺序要求、三项 SHA-256 和输出文件名

三张 PPM 都是 `480x270` P6，SHA-256 均为
`6474afe08e7ebaf0c4ae101cbd2e5a8a897c0a690a9fc21b199b6e2cb7e4cd1c`。

## 数据依赖

| Pass | Requires | Reads | Writes |
| --- | --- | --- | --- |
| `shadow` | 无 | 无 | shadow visibility |
| `scene` | `shadow` | shadow visibility | scene HDR |
| `tonemap` | `scene` | scene HDR | scene LDR |
| `ui` | 无 | 无 | UI RGBA |

真实数据边只有两条：`shadow -> scene` 和 `scene -> tonemap`。调度图只画这两条边。
`ui` 是孤立节点，和 `tonemap` 之间没有边，因此可以放在它前面、后面或与它并行。
`tonemap` 的实际输入是 `scene HDR`，所以 `scene -> tonemap` 不会被伪造成无依赖边。

`shadow` 必须完整结束后才能启动 `scene`。调度器的 `Schedule.Validate` 会把
二者放进同一 wave 的图视为非法，因此在任何合法顺序下都不会错误并行。

## 顺序无关性

UI 和 tonemap 写入不同缓冲。所有 wave 结束后才执行固定的 `Composite`：
先取 `tonemap` 产生的显示空间场景，再叠加 straight-alpha UI。合成不是调度 pass，
所以 UI 永远不会进入 `tonemap` 的输入，换序或并行不会改变亮度或颜色。

三种执行 wave：

1. `shadow -> scene -> ui -> tonemap`
2. `shadow -> scene -> tonemap -> ui`
3. `shadow -> scene -> (ui || tonemap)`

## 运行与验证

```sh
go run ./cmd/render -out artifacts
go test ./...
go test -race ./...
```

生成器会比较三份 PPM 的字节和 SHA-256；只要任一调度改变画面就直接失败。
SVG 由同一份 pass DAG 定义生成，测试同时校验 SVG 中的真实数据边数量。
