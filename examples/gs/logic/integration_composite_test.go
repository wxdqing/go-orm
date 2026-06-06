//go:build db

package logic

import (
	"testing"

	"github.com/wxdqing/go-orm/orm/drivers"
)

// EX-MY-011 / EX-PG-011 — composite_index + Find。
func TestIntegration_Composite_MySQL_GameRole_Find(t *testing.T) {
	testGameRoleFindByName(t, drivers.DriverTypeMySQL, mysqlConf())
}

func TestIntegration_Composite_Pgsql_GameRole_Find(t *testing.T) {
	testGameRoleFindByName(t, drivers.DriverTypePostgresSQL, pgsqlConf())
}

// EX-PG-012 / EX-MY — 复合主键 Save/Get/Delete。
func TestIntegration_Composite_MySQL_GameRole(t *testing.T) {
	testGameRoleCompositeCRUD(t, drivers.DriverTypeMySQL, mysqlConf())
}

func TestIntegration_Composite_Pgsql_GameRole(t *testing.T) {
	testGameRoleCompositeCRUD(t, drivers.DriverTypePostgresSQL, pgsqlConf())
}

func TestIntegration_Composite_MySQL_Lister(t *testing.T) {
	testListerCompositeCRUD(t, drivers.DriverTypeMySQL, mysqlConf())
}

func TestIntegration_Composite_Pgsql_Lister(t *testing.T) {
	testListerCompositeCRUD(t, drivers.DriverTypePostgresSQL, pgsqlConf())
}
