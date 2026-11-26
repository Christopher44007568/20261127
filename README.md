# 20261127

Repository initialized for 20261127.

## 纹理布局对比（纹理数组 vs 图集）

纯 Go 标准库实现，无第三方依赖。同一组 shader 参数 `ShaderParams{U, V, Material}`
驱动两种布局，验证边界采样正确性：

- 纹理数组：每层独立存储，层号钳制，双线性不跨层。
- 图集：单页打包，区域四周填充环绕拷贝 Gutter，双线性越界读到的是本材质
  的环绕值，既不串色也不产生可见 Gutter。

运行：

```sh
go run .
```

输出（`artifacts/`）：

- `material_balls.png` 材质球拼图：上排纹理数组、下排图集。
- `boundary_bleed.png` 边界串色图：每种材质三条横带（数组 / 图集 / 差值 x16），
  差值行纯黑表示无串色。
- `resource_table.md` 资源表：纹理对象数、绑定次数、Gutter、内存开销与校验结论。
