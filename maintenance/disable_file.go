package main

import (
	"fmt"
	"github.com/globalsign/mgo/bson"
	"github.com/ndphu/drive-manager-api/dao"
	"github.com/ndphu/drive-manager-api/entity"
)

func main() {
	dao.Init()
	defer dao.Close()
	var projects []entity.Project
	if err := dao.Project().Find(bson.M{
		"disabled": true,
	}, &projects); err != nil {
		panic(err)
	}
	for _, project := range projects {
		fmt.Println(project.Id.Hex(), project.DisplayName)
		if info, err := dao.FileIndex().UpdateAll(bson.M{"projectId": project.Id}, bson.M{
			"$set": bson.M{
				"disabled": true,
			},
		}); err != nil {
			panic(err)
		} else {
			fmt.Println("Files: Matched:", info.Matched, "Updated:", info.Updated)
		}
		//
		//if info, err := dao.DriveAccount().UpdateAll(bson.M{"projectId": project.Id}, bson.M{
		//	"$set": bson.M{
		//		"disabled": true,
		//	},
		//}); err != nil {
		//	panic(err)
		//} else {
		//	fmt.Println("Accounts: Matched:", info.Matched, "Updated:", info.Updated)
		//}
	}
}
