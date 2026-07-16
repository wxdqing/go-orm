package orm

import (
	"os"
	"strings"
	"testing"
)

func TestSQLLogsDoNotFormatConfigsOrRecords(t *testing.T) {
	cases := map[string][]string{
		"conf.go": {
			`db conf load success:%#v`,
		},
		"drivers/internal/sql/pgsql.go": {
			`current use db:  %v`,
		},
		"drivers/internal/sql/mysql.go": {
			`current use db:  %v`,
		},
		"drivers/internal/sql/base.go": {
			`GormDriver.Save execute: value [%T] [%v]`,
			`GormDriver.Find execute: cond [%T] [%v]`,
			`GormDriver.Get execute: value [%T] [%v]`,
			`GormDriver.Delete execute: value [%T] [%v]`,
		},
	}

	for path, forbidden := range cases {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, value := range forbidden {
			if strings.Contains(string(source), value) {
				t.Errorf("%s contains unsafe log format %q", path, value)
			}
		}
	}
}
