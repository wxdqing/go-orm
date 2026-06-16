# orm 包目录说明

```
orm/
├── conf.go, gorm_conf.go   # 配置结构与默认值
├── shard.go, shard_validate.go
├── handler.go, driver_context.go, interfaces.go, errors.go
├── dbinit/                 # 仅连接生命周期（Open/Close/Ping），不含表与 Handler
├── driverapi/              # 驱动接口与 Options（内部实现共享）
└── drivers/                # 进程级驱动入口，见 drivers/README.md
```

业务代码通常：

- `orm.Conf` / 分片配置 → 根包 `orm`
- `dbinit.Open` / `OpenRedis` 等 → 只要连接、不要 ORM CRUD 时
- `drivers.TryInit` / `DefaultDbDriver` → 需要 proto CRUD、AutoMigrate、Handler 时
