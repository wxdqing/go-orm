# orm 包目录说明

```
orm/
├── conf.go, gorm_conf.go   # 配置结构与默认值
├── shard.go, shard_validate.go
├── handler.go, driver_context.go, interfaces.go, errors.go
├── driverapi/              # 驱动接口与 Options（内部实现共享）
└── drivers/                # 进程级驱动入口，见 drivers/README.md
```

业务代码通常：

- `orm.Conf` / 分片配置 → 根包 `orm`
- `drivers.TryInit` / `DefaultDbDriver` → `orm/drivers`
