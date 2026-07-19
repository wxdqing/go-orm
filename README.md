# go-orm

基于 Protocol Buffers 的结构化存储组件，配合
[protoc-gen-go-orm](https://github.com/wxdqing/protoc-gen-go-orm) 生成持久化记录与编解码代码。

支持 MySQL、PostgreSQL、Redis、MongoDB 与 TcaplusDB。MySQL/PostgreSQL 提供完整查询、
计数、事务和 raw SQL 能力；其他驱动只暴露自身真实支持的能力。

## 初始化

```go
err := drivers.TryInit(ctx,
	drivers.WithConfig(conf),
	drivers.WithTables(tables),
)
if err != nil {
	return err
}
defer drivers.Close(context.Background())
```

只需要原始连接、不需要 proto CRUD、迁移和 handler 时，使用 `orm/dbinit`：

```go
db, err := dbinit.OpenRedis(ctx, conf)
if err != nil {
	return err
}
defer db.Close(context.Background())
client := db.Redis()
```

## 日志注入

go-orm 只依赖自身的 `orm.Logger` 接口，默认不输出日志。应用应在全局日志初始化完成后注入实现：

```go
orm.SetLogger(orm.LoggerFuncs{
	DebugfFunc: logger.Debugf,
	InfofFunc:  logger.Infof,
	WarnfFunc:  logger.Warnf,
	ErrorfFunc: logger.Errorf,
})
```

传入 `nil` 可恢复默认的静默日志。go-orm 不负责初始化日志实现，也不直接创建日志文件。

## 驱动能力

`driverapi.CoreDriver` 只定义所有驱动共有的生命周期与 Save/Get/Delete。可选能力使用小接口：

- `Finder`
- `Counter`
- `Transactor`
- `Pinger`
- `SQLQuerier`

兼容接口 `driverapi.Driver` 组合 Core/Finder/Counter/Transactor/Pinger。进程级
`drivers.DefaultDbDriver` 实现该接口，并额外转发 `SQLQuerier`（底层不支持时返回
`orm.ErrNotImplemented`）。`SQLQuerier.Query` 返回 `driverapi.Rows`（`*sql.Rows` 满足该接口）。
新代码持有独立驱动时，应通过类型断言检查所需能力。

Handler 的 `After` 在驱动调用返回后总会执行；其错误会与驱动错误 `Join` 后返回。

## 生命周期

`TryInit` / `Close` 会替换全局驱动。进行中的 `DefaultDbDriver` 操作会 pin 当前驱动代际，
`Close` 会等待已 pin 的调用结束后再关闭底层连接。`Query` 返回的 `*sql.Rows` 在
`Close` 前保持 pin；`QueryRow` 在首次 `Scan` 前保持 pin。若等待超时，`Close` 仍会强制
关闭底层连接并返回 `orm.ErrDbDriverCloseTimeout`。

`Close` 会先强制释放未 `Close` 的 `Query` Rows / 未 `Scan` 的 `QueryRow` pin。
同 goroutine 在 `RunInTx` 内调用 `Close` 时不等待自身 pin（事务随即失效）。
`TryInit` 的 `InitDB` 在锁外执行，慢连接不会卡住并发 `Close`。
调用方仍应避免在业务高峰期频繁 `Close`/`TryInit`。

## 分片与事务

- database 分片固定使用 `hash`，保证相同分片键稳定路由；`random` 和 `round_robin` 不可用于持久化分库配置。
- table 分片在单库事务中受支持。
- 当前 `RunInTx` 无法在开始事务前确定 database shard，因此 database 分片模式会返回 `orm.ErrNotImplemented`。
- SQL 全表 Find 必须显式使用 `orm.WithFullScan(ctx)`。

## 测试

```bash
go test ./... -count=1
go test -race ./... -count=1
go vet ./...
```

需要外部数据库的集成测试使用 `db` build tag，并通过对应环境变量提供连接配置。

目录说明见 [orm/README.md](orm/README.md) 与 [drivers/README.md](orm/drivers/README.md)。
