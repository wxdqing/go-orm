package drivers

import (
	"os"
	"testing"

	logger "gitee.com/wxdqing/logger.git"
)

func TestMain(m *testing.M) {
	logger.Init()
	os.Exit(m.Run())
}
