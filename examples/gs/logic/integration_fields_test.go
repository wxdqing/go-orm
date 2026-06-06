//go:build db

package logic

import (
	"testing"

	"github.com/wxdqing/go-orm/orm/drivers"
)

// EX-MY-010 / EX-PG-010 — FIELDS 模式 CRUD + JSON/blob 列。
func TestIntegration_Fields_MySQL_FieldsPlayer(t *testing.T) {
	testFieldsPlayerCRUD(t, drivers.DriverTypeMySQL, mysqlConf())
}

func TestIntegration_Fields_Pgsql_FieldsPlayer(t *testing.T) {
	testFieldsPlayerCRUD(t, drivers.DriverTypePostgresSQL, pgsqlConf())
}

// EX-RD-002 / EX-MG-002 — FIELDS 业务 proto 在 KV 上仍走 PAYLOAD 整包存取。
func TestIntegration_Fields_Redis_FieldsPlayer(t *testing.T) {
	testFieldsPlayerCRUD(t, drivers.DriverTypeRedis, redisConf())
}

func TestIntegration_Fields_Mongo_FieldsPlayer(t *testing.T) {
	testFieldsPlayerCRUD(t, drivers.DriverTypeMongoDB, mongoConf())
}
