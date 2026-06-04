// Package drivers 提供 ORM 进程级数据库驱动入口（TryInit / Close / DefaultDbDriver）。
//
// 目录结构：
//
//	drivers/              对外 API、生命周期、选项
//	drivers/internal/meta  表元数据注册
//	drivers/internal/codec 记录编解码
//	drivers/internal/hook  外部 Handler 注册与包装
//	drivers/internal/sql   GORM（MySQL / PostgreSQL）
//	drivers/internal/tcaplus
//	drivers/internal/kv    Redis / Mongo
//	drivers/internal/nop   空实现驱动
package drivers
