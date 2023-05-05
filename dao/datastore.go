package dao

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"os"
	"time"
)

type DataStore struct {
	Client *mongo.Client
}

type C struct {
	Name string `json:"name"`
}

func Col(name string) *C {
	return &C{
		Name: name,
	}
}

func RawCollection(name string) *mongo.Collection {
	return ds.Client.Database("drive-manager").Collection(name)
}

func RawClient() *mongo.Client {
	return ds.Client
}

var (
	ds *DataStore = nil
)

var ErrorEmptyMongoDbUri = errors.New("EmptyMongoDbUri")

func getMongoDbUri() string {
	return os.Getenv("MONGODB_URI")
}

func isSslEnabled() bool {
	return os.Getenv("MONGODB_USE_SSL") != "false"
}

func Init() error {
	if getMongoDbUri() == "" {
		return ErrorEmptyMongoDbUri
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_client, err := mongo.Connect(ctx, options.Client().ApplyURI(getMongoDbUri()))
		if err != nil {
			panic(err)
		}
		ds = &DataStore{
			Client: _client,
		}
	}
	return nil
}

func Close() {
	//if ds != nil && ds.Session != nil {
	//	ds.Session.Close()
	//}
}

func Disconnect() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := ds.Client.Disconnect(ctx); err != nil {
		panic(err)
	}
}

type TransactionCallback func(sessCtx mongo.SessionContext) (interface{}, error)

func ExecTransaction(cb TransactionCallback) (interface{}, error) {
	wc := writeconcern.New(writeconcern.WMajority())
	txnOptions := options.Transaction().SetWriteConcern(wc)
	session, err := RawClient().StartSession()
	if err != nil {
		return nil, err
	}
	defer session.EndSession(context.Background())
	return session.WithTransaction(context.Background(), cb, txnOptions)

}
