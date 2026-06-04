package drivers

import (
	"context"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestMysql_InitDB_InvalidAddrReturnsError(t *testing.T) {
	d := NewMySQLDriver().(*MysqlDriver)
	conf := testMySQLConf()
	conf.Mysql.Addr = "127.0.0.1:1"
	o := &DriverOptions{
		Type:   DriverTypeMySQL,
		Conf:   conf,
		Tables: []proto.Message{wrapperspb.String("")},
	}
	err := d.InitDB(context.Background(), o.opts())
	if err == nil {
		t.Fatal("InitDB with invalid addr: want error")
	}
}
