# Shadow-mode pipeline cache demo

这是一个只使用 Go 标准库的可复核示例。它复现“管线缓存键漏掉阴影模式”导致的错误着色器复用，并给出修复后的版本化缓存、命中记录、失效策略和像素级验证。

## 复现的缺陷

`internal/cache.LegacyKey` 只包含 shader 版本、场景版本和分辨率，**不包含 shadow mode**：

```
legacy|shader=shadow-edge-shader-2|scene=scene-ellipse-v1|320x240|<shadow-mode-missing>
```

因此先请求 `hard`、再请求 `soft` 时，legacy 缓存会命中并返回 hard 着色器。阴影边缘从柔和的半影变成硬边，同时画面左下角的模式标记仍然是 `SHADOW HARD`。

## 修复

`internal/cache.Key` 把所有会改变像素的输入都放进键中：

```
v3|shader=shadow-edge-shader-2|scene=scene-ellipse-v1|320x240|shadow=hard
v3|shader=shadow-edge-shader-2|scene=scene-ellipse-v1|320x240|shadow=soft
```

键字段包括缓存 schema 版本、shader 版本、场景版本、宽高和 `ShadowMode`。`legacy.go` 单独保留旧键，用于生成“错键”证据。

失效策略：

- `BumpSchema`：schema 版本递增，并立即 `PurgeStale` 清掉旧版本条目。
- `SetShaderVersion` / `SetSceneVersion`：shader 或场景变更时清理旧条目。
- `InvalidateMode` / `InvalidateAll`：按模式或全量失效。
- `PurgeStale`：检查 schema、shader、场景及 program 模式不变量。
- 像素校验失败：命中条目会被驱逐，然后重新编译并重建。

命中记录写入 `artifacts/report.json` 的 `fixed_cache.events` 和 `legacy_cache.events`。每条记录包含键、键摘要、请求/返回的 shadow mode、像素哈希、无缓存参考哈希、`pixels_compared`、`pixels_match` 和差异像素数。编译/查找耗时只作诊断字段，**不作为命中正确性的依据**。

## 三张 PPM

运行 `go run . -out artifacts` 生成：

- `artifacts/correct.ppm`：无缓存 soft 参考图，真值。
- `artifacts/wrong_key.ppm`：legacy 错键命中返回 hard 着色器的错误图。
- `artifacts/rebuilt.ppm`：schema bump 清空缓存后重新编译 soft 着色器得到的图。

当前生成的检查结果：

- `correct.ppm` 与 `rebuilt.ppm` 逐字节相同（SHA-256 相同）。
- `wrong_key.ppm` 与真值有 7255 / 76800 个像素不同，最大通道差 235。
- 修复后 soft 第二次请求是真正命中，且命中像素与无缓存参考 0 像素差异。
- rebuild 请求是 miss，重新编译后的像素与无缓存参考 0 像素差异。

## 运行与验证

```sh
go run . -out artifacts
go test ./...
go vet ./...
```

实现按职责拆分为：

- `internal/render`：确定性软件着色器、hard/soft 边缘、无缓存参考渲染、像素比较。
- `internal/cache`：版本化键、命中记录、失效策略、错键复现。
- `internal/ppm`：P6 二进制 PPM 写入。
- `internal/demo`：三张 PPM 和审计报告编排。

没有第三方依赖，`go.mod` 中不存在 `require`。
