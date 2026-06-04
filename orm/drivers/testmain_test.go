package drivers

import (
	"os"
	"testing"

	logger "git.wxdqing.com/sprout/logger.git"
)

func TestMain(m *testing.M) {
	logger.Init()
	os.Exit(m.Run())
}
