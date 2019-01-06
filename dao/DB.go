package dao

import (
	"fmt"
	"github.com/globalsign/mgo"
	"github.com/ndphu/drive-manager-api/config"
)

type DAO struct {
	Session *mgo.Session
	DBName  string
}

var (
	dao *DAO = nil
)

func init()  {
	conf := config.Get()
	ses, err := mgo.Dial(conf.MongoDBUri)
	if err != nil {
		fmt.Println("fail to connect to database")
		panic(err)
	} else {
		fmt.Println("database connected!")
	}

	dbName := ""

	if conf.DBName == "" {
		dbs, err := ses.DatabaseNames()
		if err != nil {
			fmt.Println("fail to connect to database")
			panic(err)
		} else {
			if len(dbs) == 0 {
				fmt.Println("no database found")
			} else {
				fmt.Println("found databases " + dbs[0])
			}
			dbName = dbs[0]
		}
	} else {
		dbName = conf.DBName
	}

	dao = &DAO{
		Session: ses,
		DBName:  dbName,
	}
}

func Collection(name string) *mgo.Collection {
	return dao.Session.DB(dao.DBName).C(name)
}

func GetSession() *mgo.Session {
	return dao.Session
}
