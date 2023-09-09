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

func dbAddUsers (users []models.UserData) error {
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