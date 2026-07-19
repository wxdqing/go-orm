package drivers

import (
	"github.com/wxdqing/go-orm/orm"
	"testing"
)

func TestConfigLoad(t *testing.T) {
	t.Run("config", func(t *testing.T) {
		m := map[string]any{
			"db": map[string]any{
				"addr":     "localhost:33061",
				"dbName":   "default-orm",
				"username": "root1",
				"password": "root2",
				"driver":   "mysqq",
			},
		}
		c := &orm.Conf{}
		if err := orm.DecodeMapToStruct(m, c); err != nil {
			t.Fatal(err)
		}
		t.Log(c)
	})
}
