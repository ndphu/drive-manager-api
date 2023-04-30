package dao

import "go.mongodb.org/mongo-driver/mongo"

func DriveAccount() *mongo.Collection {
	return RawCollection("drive_account")
}

func File() *mongo.Collection {
	return RawCollection("file")
}

func FileFavorite() *mongo.Collection {
	return RawCollection("file_favorite")
}

func FileIndex() *mongo.Collection {
	return RawCollection("file_index")
}

func FirebaseAdmin() *mongo.Collection {
	return RawCollection("firebase_admin")
}

func User() *mongo.Collection {
	return RawCollection("user")
}

func ServiceToken() *mongo.Collection {
	return RawCollection("service_token")
}

func ServiceAccountAdmin() *mongo.Collection {
	return RawCollection("service_account_admin")
}

func Project() *mongo.Collection {
	return RawCollection("project")
}

func Item() *mongo.Collection {
	return RawCollection("item")
}

func FirebaseConfig() *mongo.Collection {
	return RawCollection("firebase_config")
}

//func RunTransaction(ops []txn.Op) error {
//	return collection("transaction", func(col *mongo.Collection) error {
//		return txn.NewRunner(col).Run(ops, "", nil)
//	})
//}
