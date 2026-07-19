package orm

import (
	"fmt"
	"testing"
)

func TestSetLoggerInjectsProcessLogger(t *testing.T) {
	var message string
	SetLogger(LoggerFuncs{
		InfofFunc: func(format string, args ...any) {
			message = fmt.Sprintf(format, args...)
		},
	})
	t.Cleanup(func() { SetLogger(nil) })

	GetLogger().Infof("connected driver=%s", "mysql")
	if message != "connected driver=mysql" {
		t.Fatalf("injected logger message = %q", message)
	}
}
