package dao

import (
	"crypto/tls"
	"github.com/globalsign/mgo"
	"drive-manager-api/config"
	"log"
	"net"
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
	var session *mgo.Session
	var err error

	if conf.MongoDBUri == "" {
		session, err = mgo.Dial(conf.MongoDBUri)
	} else {
		tlsConfig := &tls.Config{}
		dialInfo, err:= mgo.ParseURL(conf.MongoDBUri)
		if err != nil {
			panic(err)
		}
		dialInfo.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
			conn, err := tls.Dial("tcp", addr.String(), tlsConfig)
			return conn, err
		}
		session, err = mgo.DialWithInfo(dialInfo)
	}

	if err != nil {
		panic(err)
	}

	dbName := ""

	if conf.DBName == "" {
		dbs, err := session.DatabaseNames()
		if err != nil {
			log.Println("fail to connect to database")
			panic(err)
		} else {
			if len(dbs) == 0 {
				log.Println("no database found")
			} else {
				log.Println("found databases " + dbs[0])
			}
			dbName = dbs[0]
		}
	} else {
		dbName = conf.DBName
	}

	dao = &DAO{
		Session: session,
		DBName:  dbName,
	}
}

func Collection(name string) *mgo.Collection {
	return dao.Session.DB(dao.DBName).C(name)
}

func GetSession() *mgo.Session {
	return dao.Session
}

func GetDB() *mgo.Database {
	return dao.Session.DB(dao.DBName)
}