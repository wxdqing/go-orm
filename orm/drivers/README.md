# drivers 包目录说明

对外仍使用 `import "…/orm/drivers"`，子目录为内部实现拆分。

```
drivers/
├── api.go              # Driver 类型别名、With* 选项、ToGorm / ToTcaplus
├── lifecycle.go        # TryInit / Close / Ping / 全局 DefaultDbDriver
├── doc.go
├── *_test.go           # 生命周期、Handler、集成测试（-tags=db）
└── internal/
    ├── meta/           # 表元数据注册（Init / GetMetaByValue）
    ├── codec/          # EncodeRecord / DecodeRecord / EnsureCtx
    ├── hook/           # HandlerRegistry 与 hook 包装
    ├── sql/            # GORM：MySQL、PostgreSQL、分片、连接池
    ├── tcaplus/        # Tcaplus 驱动与扩展 API
    ├── kv/             # Redis、Mongo 连接（CRUD 待实现）
    └── nop/            # 空实现驱动
```

`driverapi` 包（`orm/driverapi`）定义 `Driver` 接口与 `Options`，供各实现包引用以避免循环依赖。
