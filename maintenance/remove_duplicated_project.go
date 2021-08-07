package main

import (
	"fmt"
	"github.com/globalsign/mgo/bson"
	"github.com/ndphu/drive-manager-api/dao"
	"github.com/ndphu/drive-manager-api/entity"
	"github.com/ndphu/drive-manager-api/helper"
	"github.com/ndphu/drive-manager-api/service"
	"log"
)

type UserProjectLookup struct {
	Id          bson.ObjectId  `json:"id" bson:"_id"`
	DisplayName string         `json:"displayName" bson:"displayName"`
	Email       string         `json:"email" bson:"email"`
	Project     entity.Project `json:"project" bson:"project"`
}

var accountService = service.GetAccountService()
var projectService = service.GetProjectService()

type ProjectAccountLookup struct {
	Id          bson.ObjectId       `json:"id" bson:"_id"`
	ProjectId   string              `json:"projectId" bson:"projectId"`
	Owner       entity.User         `json:"owner" bson:"owner"`
	DisplayName string              `json:"displayName" bson:"displayName"`
	Account     entity.DriveAccount `json:"account" bson:"account"`
}

func main() {

	//listProjects()
	//projectId := "project-1-1546660113317"
	//deleteProjectByGoogleId(projectId)

	// project ID
	//projectId := "5c6d7d5fa88fb526241d200c"
	projectId := "61090a5da88fb508e8936cd5"

	//projectService.DeleteProject(projectId)
	if err := projectService.SyncProjectWithGoogle(projectId); err != nil {
		panic(err)
	}
}

func deleteProjectByGoogleId(projectId string) {
	var projects []ProjectAccountLookup

	if err := dao.Collection("project").Pipe([]bson.M{
		{
			"$match": bson.M{"projectId": projectId},
		},
		{

			"$lookup": bson.M{
				"from":         "drive_account",
				"localField":   "_id",
				"foreignField": "projectId",
				"as":           "account",
			},
		}, {

			"$unwind": bson.M{
				"path": "$account",
			},
		}, {

			"$lookup": bson.M{
				"from":         "user",
				"localField":   "owner",
				"foreignField": "_id",
				"as":           "owner",
			},
		}, {

			"$unwind": bson.M{
				"path": "$owner",
			},
		},
	}).All(&projects); err != nil {
		panic(err)
	}

	projectToRemove := make(map[string]bool)

	log.Println(len(projects))
	for _, project := range projects {
		pid := project.Id.Hex()
		log.Println(pid, project.DisplayName, project.Owner.Id.Hex(), project.Owner.Email, project.Owner.DisplayName)
		if _, exist := projectToRemove[pid]; !exist {
			projectToRemove[pid] = true
		}
	}

}

func listProjects() {
	var ps []UserProjectLookup
	if err := dao.Collection("user").Pipe([]bson.M{
		{
			"$lookup": bson.M{
				"from":         "project",
				"localField":   "_id",
				"foreignField": "owner",
				"as":           "project",
			},
		}, {

			"$unwind": bson.M{
				"path": "$project",
			},
		},
	}).All(&ps); err != nil {
		panic(err)
	}

	//fmt.Println(len(ps))
	m := make(map[string][]bson.ObjectId)
	for _, p := range ps {
		pid := p.Project.ProjectId
		//fmt.Println(p.Id, p.ProjectAccountLookup.Id, googleId, p.ProjectAccountLookup.DisplayName)
		if _, exist := m[pid]; !exist {
			m[pid] = make([]bson.ObjectId, 0)
		}
		m[pid] = append(m[pid], p.Project.Id)
	}
	for googleId, v := range m {
		if len(v) > 1 {
			fmt.Println(googleId)
			if googleId == "project-1-1546660113317" {
				continue
			}
			for _, id := range v {
				//fmt.Println("--> project:", id.Hex())
				if isEmptyProject(id.Hex()) {
					fmt.Println("empty project:", id.Hex())
				}
				//listFile(id)
			}
		}
	}
}

func isEmptyProject(projectId string) bool {

	accounts, err := projectService.ListAccounts(projectId)
	if err != nil {
		panic(err)
	}
	//for _, acc := range  accounts {
	//	fmt.Println(acc.ProjectId, acc.Name, acc.DisplayName)
	//}
	fmt.Println("Number of accounts:", len(accounts))

	return len(accounts) == 0
	//
	//var accList []entity.DriveAccount
	//if err := dao.Collection("drive_account").Find(bson.M{"projectId": bson.ObjectIdHex(projectId)}).All(&accList); err != nil {
	//	panic(err)
	//}
	//for _, account := range accList {
	//	if isEmpty(account) {
	//		continue
	//	}
	//	return false
	//}
	//return true
	//if len(accList) == 0 {
	//	fmt.Println("Empty project", projectId)
	//}
	//for _, acc := range accList {
	//	fmt.Println("    --> account:", acc.Id)
	//	listFile(acc)
	//}
}

func isEmpty(acc entity.DriveAccount) bool {
	ds, err := helper.GetDriveService([]byte(acc.Key))
	if err != nil {
		panic(err)
	}
	files, err := ds.ListFiles(1, 1000)
	//for _, file := range files {
	//	fmt.Println("        --> file:", file.Id, file.Name)
	//}
	return len(files) <= 0
}
