package project

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"remoteSensing/api/picture/process"
	"remoteSensing/api/response/common"
	"remoteSensing/config"
	"remoteSensing/database/sql"
	"remoteSensing/pkg/errcode"
	"remoteSensing/pkg/oss"
	"strconv"
)

type GetPicsData struct {
	Groups   []sql.Group   `json:"groups"`
	Pictures []sql.Picture `json:"pictures"`
}

func GetPictures(c *gin.Context) {
	acc, _ := c.Get("account")
	account := acc.(string)

	u, err := sql.GetUserByUsername(account)
	if err != nil {
		common.Respond(c, errcode.UnauthorizedUserNotFound, gin.H{})
		return
	}

	pID := c.Param("id")
	projectID, _ := strconv.Atoi(pID)

	isExist, err := sql.IsExist(uint64(projectID), u.ID)
	if !isExist || err != nil {
		common.Respond(c, errcode.AccountProjectError, gin.H{})
		return
	}

	folderName := fmt.Sprintf("project/%d/%d", u.ID, projectID)
	names, err := oss.GetAllNames(folderName)
	if err != nil {
		common.Respond(c, errcode.GetFolderError, gin.H{})
		return
	}

	groupInfos, err := sql.GetGroups(uint64(projectID))
	if err != nil {
		common.Respond(c, errcode.ServerError, gin.H{})
		return
	}
	groups := make([]sql.Group, len(groupInfos))
	groupMap := make(map[uint64]int, len(groups))
	for i := 0; i < len(groups); i++ {
		groups[i] = sql.Group{
			ID:       groupInfos[i].ID,
			Name:     groupInfos[i].Name,
			Type:     groupInfos[i].Type,
			Info:     groupInfos[i].Info,
			Pictures: []sql.Picture{},
		}
		if groupInfos[i].Type == process.CDType {
			groups[i].Pictures = make([]sql.Picture, 3)
		} else if groupInfos[i].Type == process.OverallType {
			num, err := sql.GetOverallPicturesNum(groupInfos[i].ID)
			if err != nil {
				common.Respond(c, errcode.ServerError, gin.H{})
				return
			}
			groups[i].Pictures = make([]sql.Picture, num+3)
		} else {
			groups[i].Pictures = make([]sql.Picture, 2)
		}
		groupMap[groups[i].ID] = i
	}

	//fmt.Println(names)

	uuids := make([]string, len(names))
	for i := 0; i < len(names); i++ {
		uuids[i] = names[i][len(folderName)+1 : len(names[i])-4]
	}

	//fmt.Println(uuids)

	pics, err := sql.GetPictures(uuids)
	if err != nil {
		common.Respond(c, errcode.ServerError, gin.H{})
		return
	}

	restPics := []sql.Picture{}

	//fmt.Println(pics)

	for i := 0; i < len(pics); i++ {
		url := config.ProjectPath + folderName + "/" + pics[i].UUID + ".jpg"

		pic := sql.Picture{
			UUID: pics[i].UUID,
			Name: pics[i].Name,
			URL:  url,
		}
		picIndex := pics[i].Type - 1

		if index, ok := groupMap[pics[i].GroupID]; ok {
			groups[index].Pictures[picIndex] = pic
		} else {
			restPics = append(restPics, pic)
		}
	}

	for i := 0; i < len(groups); i++ {
		for j := 0; j < len(groups[i].Pictures); j++ {
			if groups[i].Pictures[j].UUID == "" {
				groups[i].Pictures = append(groups[i].Pictures[:j], groups[i].Pictures[j+1:]...)
				j--
			}
		}
	}

	data := GetPicsData{
		Groups:   groups,
		Pictures: restPics,
	}

	common.Respond(c, errcode.Success, data)
}
