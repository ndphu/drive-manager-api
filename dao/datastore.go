package dao

import (
	"crypto/tls"
	"errors"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"net"
	"os"
)

type DataStore struct {
	Session *mgo.Session
}

type C struct {
	Name string `json:"name"`
}

func Col(name string) *C {
	return &C{
		Name: name,
	}
}

type CollectionFunc func(col *mgo.Collection) error

func collection(name string, cb CollectionFunc) error {
	session := ds.Session.Copy()
	defer session.Close()
	return cb(session.DB("drive-manager").C(name))
}

func (c *C) Pipe(pipe []bson.M, result interface{}) error {
	return collection(c.Name, func(c *mgo.Collection) error {
		return c.Pipe(pipe).All(result)
	})
}

func (c *C) Insert(docs ...interface{}) error {
	return collection(c.Name, func(c *mgo.Collection) error {
		return c.Insert(docs...)
	})
}

func (c *C) FindId(id bson.ObjectId, result interface{}) error {
	return collection(c.Name, func(col *mgo.Collection) error {
		return col.FindId(id).One(result)
	})
}

func (c *C) FindAll(result interface{}) error {
	return collection(c.Name, func(col *mgo.Collection) error {
		return col.Find(nil).All(result)
	})
}

func (c *C) Find(filter bson.M, result interface{}) error {
	return collection(c.Name, func(col *mgo.Collection) error {
		return col.Find(filter).All(&result)
	})
}

func (c *C) Template(cb CollectionFunc) error {
	return collection(c.Name, cb)
}

func (c *C) PipeOne(pipe []bson.M, result interface{}) error {
	return c.Template(func(c *mgo.Collection) error {
		return c.Pipe(pipe).One(result)
	})
}

func (c *C) FindOne(filter bson.M, result interface{}) error {
	return c.Template(func(col *mgo.Collection) error {
		return col.Find(filter).One(&result)
	})
}

func (c *C) UpdateId(id bson.ObjectId, e interface{}) error {
	return c.Template(func(col *mgo.Collection) error {
		return col.UpdateId(id, e)
	})
}

func (c *C) Update(selector bson.M, update bson.M) error {
	return c.Template(func(col *mgo.Collection) error {
		return col.Update(selector, update)
	})
}

func (c *C) Count(filter bson.M) (int, error) {
	result := 0
	err := c.Template(func(col *mgo.Collection) error {
		count, err := col.Find(filter).Count()
		if err != nil {
			return err
		}
		result = count
		return nil
	})
	return result, err
}

func (c *C) RemoveAll(filter bson.M) (*mgo.ChangeInfo, error) {
	var info *mgo.ChangeInfo
	err := c.Template(func(col *mgo.Collection) error {
		_info, err := col.RemoveAll(filter)
		info = _info
		return err
	})
	return info, err
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
	var session *mgo.Session

	if getMongoDbUri() == "" {
		return ErrorEmptyMongoDbUri
	} else {
		if isSslEnabled() {
			tlsConfig := &tls.Config{}
			dialInfo, err := mgo.ParseURL(getMongoDbUri())
			if err != nil {
				return err
			}
			dialInfo.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
				conn, err := tls.Dial("tcp", addr.String(), tlsConfig)
				return conn, err
			}
			session, err = mgo.DialWithInfo(dialInfo)
			if err != nil {
				return err
			}
		} else {
			//dialInfo, err := mgo.ParseURL(getMongoDbUri())
			s, err := mgo.Dial(getMongoDbUri())
			if err != nil {
				return err
			}
			session = s
		}
	}
	ds = &DataStore{
		Session: session,
	}
	return nil
}

func Close() {
	if ds != nil && ds.Session != nil {
		ds.Session.Close()
	}
}
