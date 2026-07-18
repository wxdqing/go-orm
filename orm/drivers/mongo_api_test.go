package drivers

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

func TestDriverMongoReturnsV2Client(t *testing.T) {
	var client *mongo.Client = DriverMongo(nil)
	if client != nil {
		t.Fatal("DriverMongo(nil) returned a client")
	}
}
