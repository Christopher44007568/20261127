# Triplanar Cliff Lab

只用 Go 标准库实现的世界空间三平面材质与 CPU 软光栅场景。它不依赖模型 UV，
直接用世界坐标投影 `X/Y/Z` 三个平面，并用归一化法线权重混合，适合倾斜悬崖、
洞顶、底面和负缩放镜像实例。

## 运行

```sh
go test ./...
go run .
```

默认生成 960×720、2× SSAA 的四张 PNG 和诊断数据：

- `artifacts/rock_wall.png`：倾斜、带层理和位移的岩壁。
- `artifacts/cave_ceiling.png`：仰视拱形洞顶。
- `artifacts/underside.png`：仰视悬挑岩石底面。
- `artifacts/mirrored_instance.png`：非对称悬崖与关于 `X=0` 精确反射的镜像实例。
- `artifacts/metrics.json`：覆盖率、亮度、权重误差和白纹理接缝峰值。

常用参数：

```sh
go run . -out artifacts -scale 0.55 -hardness 4 -width 960 -height 720 -aa 2
go run . -hardness 1.5   # 更软、更宽的投影混合区
go run . -hardness 10    # 更硬、更窄的投影过渡
```

## 三平面公式

法线先归一化并取绝对值，硬度 `h` 是权重指数：

```text
(wx, wy, wz) = pow(abs(N), h)
(wx, wy, wz) /= wx + wy + wz
Color = wx * TextureX + wy * TextureY + wz * TextureZ
```

权重之和严格为 `1`，混合是凸组合而不是三次采样相加。因此在三种投影共同覆盖的
位置，白纹理的峰值仍为 `1`，不会出现加法式过亮接缝。测试用随机法线锁定这一约束。

世界尺度 `s = WorldScale` 直接作用于世界坐标，`1/s` 个世界单位对应一张纹理周期：

| 投影 | U | V |
| --- | --- | --- |
| X | `Z*s` | `Y*s` |
| Y | `X*s` | `Z*s` |
| Z | `X*s` | `Y*s` |

`X/Z` 两个侧向投影始终使用同一个世界重力坐标 `Y*s`，所以倾斜面只会改变法线权重，
不会把重力方向的层理尺度拉长或压缩。默认使用统一尺度，避免轴向比例破坏该约束。

镜像或负缩放实例的位置直接在世界空间投影；法线按缩放矩阵的逆转置变换。
`MirrorMeshX` 同时反射位置和法线，保证镜像实例的光照和权重仍然连续。

## 实现结构

- `triplanar.go`：尺度、法线权重、硬度和凸组合采样。
- `texture.go`：零依赖可平铺程序化岩壁纹理与双线性采样。
- `mipmap.go`：标准库生成的 box mipmap，配合屏幕导数选择 LOD，减少透视远处摩尔纹。
- `mesh.go`：参数网格、世界噪声、TRS/负缩放法线变换和精确镜像。
- `raster.go`：透视相机、深度缓冲、透视正确的世界坐标/法线插值。
- `render.go`：线性光照、SSAA、Reinhard 色调映射和 sRGB 输出。
- `scenes.go`：四个指定场景。
- `triplanar_test.go`：权重、接缝能量、重力尺度、镜像和渲染覆盖测试。

## 产物预览

![岩壁](artifacts/rock_wall.png)

![洞顶](artifacts/cave_ceiling.png)

![底面](artifacts/underside.png)

![镜像实例](artifacts/mirrored_instance.png)
