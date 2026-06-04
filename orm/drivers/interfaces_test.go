package drivers

import (
	"github.com/wxdqing/go-orm/orm"
	logger "git.wxdqing.com/sprout/logger.git"
	"testing"
)

func TestConfigLoad(t *testing.T) {
	logger.Init()
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
			logger.Fatalf("orm decode conf map to struct fail: %s", err.Error())
		}
		t.Log(c)
	})
}
