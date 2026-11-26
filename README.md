# 清漆湿面能量分层渲染

纯 Go 标准库实现的离线 CPU 渲染器。表面由**底层漫反射**和**单层清漆环境反射**
组成，不额外叠加底层宽高光，避免两层高光把饱和底色冲白。

## 能量模型

对视线与法线的夹角余弦 `NdotV`，清漆 Fresnel 与能量分配为：

```text
F  = F0 + (1 - F0) * (1 - NdotV)^5
Fc = min(clearcoatWeight * F, energyCap)

baseDiffuse = albedo * diffuseLight * (1 - Fc)
coatReflect = reflectedEnvironment * Fc
final       = baseDiffuse + coatReflect
```

- 默认 `F0 = 0.04`，默认掠射能量上限 `energyCap = 0.85`。
- `Fc` 同时控制反射能量和底层剩余能量，二者之和恒为 1。
- 环境反射值、漫反射光照和底色均限制在 `[0,1]`，因此单像素总能量不超过 1。
- 清漆粗糙度只改变环境反射的滤波锥角，不增加额外高光能量。
- `clearcoatWeight == 0` 时 `Fc == 0`，输出严格等于原始漫反射底层。

## 运行

```bash
go run ./cmd/clearcoat
go test ./...
```

可指定输出目录：

```bash
go run ./cmd/clearcoat -out out
```

项目仅使用 Go 标准库，无第三方依赖。

## 渲染结果

- `out/dry_wet.png`：相同底色下，清漆权重 0 与 1 的干湿对照。
- `out/reflection_separation.png`：底层漫反射、湿面合成、清漆反射三栏分离图。
- `out/roughness_gradient.png`：单球面上从光滑到粗糙的清漆反射渐变。

实现位于 `render/`：`brdf.go` 负责 Fresnel 与能量分配，
`environment.go` 负责程序化软盒环境和粗糙度滤波，`renderer.go` 负责光线求交与成像。
