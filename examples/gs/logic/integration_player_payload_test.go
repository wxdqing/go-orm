//go:build db

package logic

import (
	"testing"

	"github.com/wxdqing/go-orm/orm/drivers"
)

// EX-MY-013 / EX-PG — 复杂嵌套 Player PAYLOAD 往返。
func TestIntegration_PlayerPayload_MySQL(t *testing.T) {
	testPlayerPayloadCRUD(t, drivers.DriverTypeMySQL, mysqlConf(), sqlCoverageTableNames()...)
}

func TestIntegration_PlayerPayload_Pgsql(t *testing.T) {
	testPlayerPayloadCRUD(t, drivers.DriverTypePostgresSQL, pgsqlConf(), sqlCoverageTableNames()...)
}

// EX-RD-002/003、EX-MG-002/003 — KV Player PAYLOAD。
func TestIntegration_PlayerPayload_Redis(t *testing.T) {
	testPlayerPayloadCRUD(t, drivers.DriverTypeRedis, redisConf(), kvCoverageTableNames()...)
}

func TestIntegration_PlayerPayload_Mongo(t *testing.T) {
	testPlayerPayloadCRUD(t, drivers.DriverTypeMongoDB, mongoConf(), kvCoverageTableNames()...)
}
