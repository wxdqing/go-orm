package testenv

import (
	"os"
	"strings"
	"testing"
)

func TestMongoURI_DefaultMatchesDockerCompose(t *testing.T) {
	os.Unsetenv("ORM_TEST_MONGO_URI")
	os.Unsetenv("ORM_TEST_MONGO_USER")
	os.Unsetenv("ORM_TEST_MONGO_PASSWORD")
	os.Unsetenv("ORM_TEST_MONGO_ADDR")
	os.Unsetenv("ORM_TEST_MONGO_AUTH_DB")

	uri := MongoURI()
	if !strings.Contains(uri, MongoDefaultUser) {
		t.Fatalf("uri should contain user %q: %s", MongoDefaultUser, uri)
	}
	if !strings.Contains(uri, "authSource="+MongoDefaultAuthDB) {
		t.Fatalf("uri should use authSource=%s: %s", MongoDefaultAuthDB, uri)
	}
	if !strings.Contains(uri, MongoDefaultAddr) {
		t.Fatalf("uri should contain addr %q: %s", MongoDefaultAddr, uri)
	}
}
