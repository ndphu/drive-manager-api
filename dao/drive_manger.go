package dao

import (
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/txn"
	"go.mongodb.org/mongo-driver/mongo"
)

func DriveAccount() *C {
	return Col("drive_account")
}

func File() *C {
	return Col("file")
}

func FileFavorite() *C {
	return Col("file_favorite")
}

func FileIndex() *C {
	return Col("file_index")
}

func FirebaseAdmin() *C {
	return Col("firebase_admin")
}

func User() *C {
	return Col("user")
}

func ServiceToken() *C {
	return Col("service_token")
}

func ServiceAccountAdmin() *C {
	return Col("service_account_admin")
}

func Project() *C {
	return Col("project")
}

func Item() *C {
	return Col("item")
}

func FirebaseConfig() *C {
	return Col("firebase_config")
}

func RunTransaction(ops []txn.Op) error {
	return collection("transaction", func(col *mongo.Collection) error {
		return txn.NewRunner(col).Run(ops, "", nil)
	})
}
