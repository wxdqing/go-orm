//go:build bdd

package bdd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cucumber/godog"
)

func TestGodog(t *testing.T) {
	featureDir := filepath.Join("..", "docs", "orm", "bdd")
	if _, err := os.Stat(featureDir); err != nil {
		t.Skipf("feature dir missing: %v", err)
	}
	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{featureDir},
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("godog suite failed")
	}
}
