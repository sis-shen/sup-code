# Cordis 内核 Go API 使用说明

> 适用版本：Sup Harness 2.0（`v2.0.0-alpha.1`）
> 包路径：`github.com/supcode/supcode/core`
> 设计决策：见 [`adr/ADR-0001-cordis-kernel.md`](adr/ADR-0001-cordis-kernel.md)

## 1. 创建内核与作用域

```go
k := core.New()
root := k.Root()
```

`Context` 是服务容器与生命周期作用域。`Fork` 派生子作用域并继承服务；`Isolate` 额外阻断向父/内核解析服务。

```go
child := root.Fork("session-1")   // 继承 root 可见的服务
private := root.Isolate("subagent") // 看不到 root/内核的服务
```

## 2. 提供与消费服务

```go
core.Provide(child, "clock", myClock)          // 注册（可逆）
clock := core.Use[Clock](child, "clock")        // 缺失/类型不符会 panic
c, ok := core.MaybeUse[Clock](child, "clock")   // 安全查询
child.Has("clock")                              // 是否存在
```

`Provide` 返回 `Disposer`，并**自动登记**到该作用域；作用域 Dispose 时服务被撤销。

## 3. 注册可逆副作用

```go
err := child.Effect(func() (core.Disposer, error) {
    conn := openConn()
    return func() error { return conn.Close() }, nil
})
```

`Dispose` 逆序回放；重复 `Dispose` 幂等。

## 4. 事件五模式

```go
child.On("tools/pre-execute", func(ctx context.Context, payload any, next core.Next) (any, error) {
    if blocked(payload) {
        return nil, errBlocked      // 短路 waterfall
    }
    res, err := next(payload)       // 继续链
    if err != nil { return nil, err }
    return postProcess(res), nil
})

core.Emit(ctx, k, "audit", payload)                 // 通知
resp, err := core.Waterfall[Req, Resp](ctx, k, "req", req) // 环绕中间件
err = core.Parallel(ctx, k, "telemetry", payload)    // 并发扇出
resp, err = core.Serial[Resp](ctx, k, "decide", payload)   // 按序
resp, hit, err := core.Bail[Resp](ctx, k, "pick", payload) // 首个非 nil 短路
```

- 监听器顺序 = 注册顺序；`core.Once()` 使其首次触发后自动移除。
- `Context.On` 注册的监听器随作用域 Dispose 自动移除。

## 5. 定义与加载插件

```go
p := core.Plugin{
    Name:     "plugin-clock",
    Inject:   []string{"config"},
    Provides: []string{"clock"},
    Config:   func() any { return map[string]any{"tz": "UTC"} },
    Apply: func(ctx *core.Context) error {
        cfg := ctx.Config()                 // 本插件的配置树
        core.Provide(ctx, "clock", NewClock(cfg))
        ctx.On("tick", onTick)
        return nil
    },
}

k.Load(k.Root(), p)          // 校验 inject → Fork → Apply；失败回滚
k.LoadPlugins(k.Root(), []core.Plugin{p2, p1}) // 依 Provides/Inject 拓扑加载
k.Unload("plugin-clock")     // 逆序回放副作用
k.Reload(k.Root(), "plugin-clock")
k.Shutdown(ctx)              // 逆序卸载全部插件
```

## 6. 运行演示

```bash
go test ./core/... ./plugin/demo/... -race
```

`plugin/demo` 演示了 `Provide/inject`、`demo/add` 事件、可逆 effect 与作用域卸载。
