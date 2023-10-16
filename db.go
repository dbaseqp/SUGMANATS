package main

import (
	// "fmt"

	// "github.com/gin-gonic/gin"
	// "gorm.io/gorm/clause"

	"sugmanats/models"
)

func dbGetBoxes () ([]models.Box, error) {
	var boxes []models.Box
	
	subquery := db.Table("boxes").Select("id,MAX(timestamp)").Group("ip")
	result := db.Table("boxes").Joins("INNER JOIN (?) as grouped on boxes.id = grouped.id", subquery).Find(&boxes)

	if result.Error != nil {
		return nil, result.Error
	}

	return boxes, nil
}

func dbGetPorts () (map[uint][]models.Port, error) {
	var ports []models.Port
	
	subquery := db.Table("ports").Select("id,MAX(timestamp)").Group("box_id,port")
	result := db.Table("ports").Joins("INNER JOIN (?) as grouped on ports.id = grouped.id", subquery).Find(&ports)

	if result.Error != nil {
		return nil, result.Error
	}
	portMap := map[uint][]models.Port{}
	for _, port := range ports {
		portMap[port.BoxID] = append(portMap[port.BoxID], port)
	}
	return portMap, nil
}

func dbGetUser (id uint) (models.UserData, error) {
	var user models.UserData
	
	result := db.First(&user, id)

	if result.Error != nil {
		return models.UserData{}, result.Error
	}

	return user, nil
}

func dbGetUsers () (map[uint]models.UserData, error) {
	var users []models.UserData
	
	result := db.Table("user_data").Find(&users)

	if result.Error != nil {
		return nil, result.Error
	}

	userMap := map[uint]models.UserData{}
	for _, user := range users {
		userMap[user.ID] = user
	}
	return userMap, nil
}

func dbGetCredentials() ([]models.Credential, error) {
	var credentials []models.Credential
	
	result := db.Table("credentials").Find(&credentials)

	if result.Error != nil {
		return nil, result.Error
	}

	return credentials, nil
}

func dbAddCredential(credential *models.Credential) error {
	result := db.Create(&credential)

	if result.Error != nil {
		return result.Error
	}

	return nil
}

func dbEditCredential(credential *models.Credential) error {
	result := db.Model(credential).Select("username", "password", "note").Updates(&credential)

	if result.Error != nil {
		return result.Error
	}

	return nil
}

func dbDeleteCredential(id uint) error {
	result := db.Delete(&models.Credential{}, id)

	if result.Error != nil {
		return result.Error
	}

	return nil
}

func dbAddUsers(users []models.UserData) error {
	for _, user := range users {
		if db.Model(&user).Where("name = ?", user.Name).Updates(&user).RowsAffected == 0 {
			db.Create(&user)
		}
	}
	
	return nil
}

func dbUpdateBoxDetails(box *models.Box) error {
	subquery := db.Model(box).Select("id,MAX(timestamp)").Group("ip")
	result := db.Model(box).Select("usershells","rootshells","claimer_id").Joins("INNER JOIN (?) as grouped on boxes.id = grouped.id", subquery).Updates(box)
	if result.Error != nil {	
		return result.Error
	}
	return nil
}

func dbUpdateBoxNote(box *models.Box) error {
	subquery := db.Model(box).Select("id,MAX(timestamp)").Group("ip")
	result := db.Model(box).Select("note").Joins("INNER JOIN (?) as grouped on boxes.id = grouped.id", subquery).Updates(box)
	if result.Error != nil {	
		return result.Error
	}
	return nil
}

func dbEditSettings(user *models.UserData) error {
	result := db.Model(user).Select("color").Updates(&user)

	if result.Error != nil {
		return result.Error
	}

	return nil
}

func dbPropagateData(box models.Box) (models.Box, error) {
	var oldBox models.Box
	
	// see if IP exists
	if err := db.Table("boxes").Where("ip = (?)", box.IP).First(&models.Box{}).Error; err != nil {
		return box, nil
	}

	subquery := db.Table("boxes").Select("id,ip,MAX(timestamp)").Group("ip")
	result := db.Table("boxes").Joins("INNER JOIN (?) as grouped on boxes.id = grouped.id", subquery).Where("boxes.IP = ?", box.IP).First(&oldBox)

	if result.Error != nil {
		return models.Box{}, result.Error
	}
	// these shouldn't change when merging scans
	box.ClaimerID = oldBox.ClaimerID
	box.Rootshells = oldBox.Rootshells
	box.Usershells = oldBox.Usershells
	box.Note = oldBox.Note

	return box, nil
}

func dbGetTasks () ([]models.Task, error) {
	var tasks []models.Task
	
	result := db.Preload("Assignee").Find(&tasks)

	if result.Error != nil {
		return nil, result.Error
	}

	return tasks, nil
}

func dbGetMyTasks (id uint) ([]models.Task, error) {
	var tasks []models.Task
	
	result := db.Preload("tasks").Find(&tasks)

	if result.Error != nil {
		return nil, result.Error
	}

	return tasks, nil
}

func dbAddTask (task *models.Task) error {
	result := db.Create(&task)

	if result.Error != nil {
		return result.Error
	}

	return nil
}

func dbEditTask(task *models.Task) error {
	result := db.Model(task).Select("note", "due_time", "status", "assignee_id").Updates(&task)

	if result.Error != nil {
		return result.Error
	}

	return nil
}

func dbDeleteTask(id uint) error {
	result := db.Delete(&models.Task{}, id)

	if result.Error != nil {
		return result.Error
	}

	return nil
}