package dao

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

type CollectionFunc func(col *mongo.Collection) error

func collection(name string, cb CollectionFunc) error {
	return cb(ds.Client.Database("driver-manager").Collection(name))
}

func (c *C) Pipe(matchStage, groupStage bson.D, result interface{}) error {
	return c.Template(func(c *mongo.Collection) error {
		cursor, err := c.Aggregate(context.Background(), mongo.Pipeline{groupStage, matchStage})
		if err != nil {
			return err
		}
		return cursor.All(context.Background(), result)
	})
}

func (c *C) Insert(docs ...interface{}) error {
	return c.Template(func(c *mongo.Collection) error {
		_, err := c.InsertMany(context.Background(), docs)
		return err
	})
}

func (c *C) FindId(id primitive.ObjectID, result interface{}) error {
	return c.Template(func(col *mongo.Collection) error {
		return col.FindOne(context.Background(), bson.D{{"_id", id}}).Decode(result)
	})
}

func (c *C) FindAll(result interface{}) error {
	return c.Template(func(col *mongo.Collection) error {
		find, err := col.Find(context.Background(), bson.D{{}})
		if err != nil {
			return err
		}
		return find.All(context.Background(), result)
	})
}

func (c *C) Find(filter bson.D, result interface{}) error {
	return c.Template(func(col *mongo.Collection) error {
		find, err := col.Find(context.Background(), filter)
		if err != nil {
			return err
		}
		return find.All(context.Background(), result)
	})
}

func (c *C) Template(cb CollectionFunc) error {
	return collection(c.Name, cb)
}

func (c *C) PipeOne(matchStage, groupStage bson.D, result interface{}) error {
	return c.Template(func(c *mongo.Collection) error {
		cursor, err := c.Aggregate(context.Background(), mongo.Pipeline{matchStage, groupStage})
		if err != nil {
			return err
		}
		return cursor.All(context.Background(), result)
	})
}

func (c *C) Count(filter bson.D) (int64, error) {
	var result int64 = 0
	err := c.Template(func(col *mongo.Collection) error {
		count, err := col.CountDocuments(context.Background(), filter)
		if err != nil {
			return err
		}
		result = count
		return nil
	})
	return result, err
}

func (c *C) RemoveAll(filter bson.D) error {
	return c.Template(func(col *mongo.Collection) error {
		_, err := col.DeleteMany(context.Background(), filter)
		return err
	})
}

func (c *C) UpdateAll(filter bson.D, update bson.D) error {
	return c.Template(func(col *mongo.Collection) error {
		_, err := col.UpdateMany(context.Background(), filter, update)
		return err
	})
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
