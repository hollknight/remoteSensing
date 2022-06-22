package process

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"remoteSensing/api/response/common"
	"remoteSensing/config"
	"remoteSensing/database/sql"
	"remoteSensing/global"
	"remoteSensing/pkg/errcode"
	"remoteSensing/pkg/resource"
)

const OverallType = 1

type OverallRequest struct {
	ProjectID  uint64 `json:"projectID"`
	OriginUUID string `json:"originUUID"`
}

type OverallData struct {
	OA PicData     `json:"oa"`
	GS PicData     `json:"gs"`
	OD PicData     `json:"od"`
	CD [][]PicData `json:"cd"`
}

type PicData struct {
	UUID string `json:"uuid"`
	URL  string `json:"url"`
	Name string `json:"name"`
}

type OverallInfo struct {
	Mark []int `json:"mark"`
}

func ProcessPicOverall(c *gin.Context) {
	acc, _ := c.Get("account")
	account := acc.(string)

	request := new(OverallRequest)
	err := c.ShouldBindJSON(&request)
	if err != nil {
		common.Respond(c, errcode.ParamError, gin.H{})
		return
	}

	u, err := sql.GetUserByUsername(account)
	if err != nil {
		common.Respond(c, errcode.AccountProjectError, gin.H{})
		return
	}

	originUUID := request.OriginUUID
	originName := originUUID + ".jpg"
	ok, err := resource.IsExist(config.LocalPicPath + originName)
	if !ok || err != nil {
		common.Respond(c, errcode.ServerError, gin.H{})
		return
	}

	projectID := request.ProjectID

	groupInfos, err := sql.GetOverallGroups(originUUID)
	if err != nil {
		common.Respond(c, errcode.ServerError, gin.H{})
		fmt.Println(111)
		return
	}

	var oa, gs, od PicData
	var cd [][]PicData

	err = global.GLOBAL_DB.Transaction(func(tx *gorm.DB) error {
		// 创建 group
		groupID, txErr := sql.AddGroup(tx, projectID, "综合分析台", OverallType, nil)
		if txErr != nil {
			return txErr
		}

		mark := make([]int, 4)

		var cdBase int8 = 4
		for i := 0; i < len(groupInfos); i++ {
			picInfos, txErr := sql.GetPicturesByGroupID(groupInfos[i].ID)
			if txErr != nil {
				return txErr
			}

			fileName := fmt.Sprintf("project/%d/%d/%s.jpg", u.ID, projectID, picInfos[0].UUID)
			url := fmt.Sprintf("%s%s", config.ProjectPath, fileName)
			uuid := picInfos[0].UUID
			name := picInfos[0].Name

			mark[groupInfos[i].Type-2]++

			if groupInfos[i].Type == OAType {
				oa = PicData{
					UUID: uuid,
					URL:  url,
					Name: name,
				}
				txErr = sql.AddPicture(tx, uuid, name, projectID, groupID, 1)
				if txErr != nil {
					return txErr
				}
			} else if groupInfos[i].Type == GSType {
				gs = PicData{
					UUID: uuid,
					URL:  url,
					Name: name,
				}
				txErr = sql.AddPicture(tx, uuid, name, projectID, groupID, 2)
				if txErr != nil {
					return txErr
				}
			} else if groupInfos[i].Type == ODType {
				od = PicData{
					UUID: uuid,
					URL:  url,
					Name: name,
				}
				txErr = sql.AddPicture(tx, uuid, name, projectID, groupID, 3)
				if txErr != nil {
					return txErr
				}
			} else if groupInfos[i].Type == CDType {
				var resPic, oldPic, newPic PicData

				for j := 0; j < len(picInfos); j++ {
					fileName = fmt.Sprintf("project/%d/%d/%s.jpg", u.ID, projectID, picInfos[j].UUID)
					url = fmt.Sprintf("%s%s", config.ProjectPath, fileName)
					uuid = picInfos[j].UUID
					name = picInfos[j].Name

					if picInfos[j].Type == 1 {
						resPic = PicData{
							UUID: uuid,
							URL:  url,
							Name: name,
						}
						txErr = sql.AddPicture(tx, uuid, name, projectID, groupID, cdBase)
						if txErr != nil {
							return txErr
						}
					} else if picInfos[j].Type == 2 {
						oldPic = PicData{
							UUID: uuid,
							URL:  url,
							Name: name,
						}
						txErr = sql.AddPicture(tx, uuid, name, projectID, groupID, cdBase+2)
						if txErr != nil {
							return txErr
						}
					} else if picInfos[j].Type == 3 {
						newPic = PicData{
							UUID: uuid,
							URL:  url,
							Name: name,
						}
						txErr = sql.AddPicture(tx, uuid, name, projectID, groupID, cdBase+1)
						if txErr != nil {
							return txErr
						}
					}
				}
				cdBase += 3

				cd = append(cd, []PicData{resPic, newPic, oldPic})
			}
		}

		overallInfo := OverallInfo{Mark: mark}
		infoJson, _ := json.Marshal(overallInfo)

		txErr = sql.UpdateGroupInfo(tx, projectID, groupID, infoJson)
		if txErr != nil {
			return txErr
		}

		return nil
	})
	if err != nil {
		common.Respond(c, errcode.ServerError, gin.H{})
		fmt.Println(err)
		return
	}

	data := OverallData{
		OA: oa,
		GS: gs,
		OD: od,
		CD: cd,
	}
	common.Respond(c, errcode.Success, data)
}
