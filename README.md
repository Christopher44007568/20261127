# Go 裁剪蒙版与裁剪栈

零第三方依赖的 Go 示例：用标准库实现圆形、星形、矢量文本蒙版，以及裁剪栈、逆裁剪、蒙版交集和真正的 alpha 镂空。

## 运行

```sh
go run .
go test ./...
```

默认写入 `outputs/`，也可以指定目录：

```sh
go run . -out /path/to/pngs
```

## 产物

- `outputs/circle_mask.png`、`outputs/star_mask.png`、`outputs/text_mask.png`：三种独立抗锯齿 alpha 蒙版。
- `outputs/alpha_*_mask.png`：对应的白覆盖、黑遮挡灰度蒙版。
- `outputs/final_composite.png`：总览合成图，背景和所有内部洞都是真实透明像素。
- `outputs/final_preview_checker.png`：把合成图放到棋盘格上的预览。
- `outputs/inverse_circle_mask.png`：圆内逆裁剪形成的环形区域。
- `outputs/intersection_mask.png`：圆形与星形的交叉区域。
- `outputs/triple_overlay.png`：圆、星、`GO` 文本三层蒙版在同一坐标上的叠加。
- `outputs/triple_clip_stack.png`：圆 ∩ 星 ∩ 文本的裁剪栈结果。

## 实现要点

- `Mask` 内部使用 `image.Alpha` 保存覆盖率；`0` 表示完全镂空，`255` 表示完全保留。
- 圆、多边形、矢量字形都通过子像素超采样生成边缘覆盖率，因此边缘具有抗锯齿过渡。
- `Canvas.Push(mask, mode)` / `Canvas.Pop()` 构成裁剪栈：
  - `ClipIntersect`：父裁剪区与新蒙版相乘，即交叉保留。
  - `ClipInverse`：父裁剪区与 `mask.Inverse()` 相乘，实现逆裁剪。
  - `ClipReplace`：用于独立图层。
- `Intersect(a, b)` 使用 alpha 乘法组合覆盖率。
- `Canvas.Cut` 实现 destination-out；完全挖空的位置会写成 `RGBA(0,0,0,0)`，不是白色或其他背景色。
- 内层 `Pop` 后会恢复父裁剪蒙版，因此后续兄弟对象不会继续受到嵌套蒙版限制；父裁剪仍会约束它们。

`main_test.go` 对透明洞、抗锯齿中间覆盖率、逆蒙版、交集、`Pop` 隔离和最终 PNG 像素都做了断言。
