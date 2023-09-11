package models

import (
	"time"
)

type Box struct {
	ID   		uint
	Status		string
	Hostname 	string
	IP   		string
	Timestamp	time.Time `gorm:"column:timestamp;default:CURRENT_TIMESTAMP"`
	Usershells	int
	Rootshells	int
	Note		string
	ClaimerID	uint
	Claimer		UserData
}

type UserData struct {
	ID          uint
	Name		string
	Pw			string
	Color		string
	Tasks		[]Task `gorm:"foreignKey:AssigneeID"`
}

type Port struct {
	ID 			uint
	BoxID		uint
	Box			Box
	Port		string
	Protocol	string
	State		string
	Service		string
	Tunnel		string
	Version		string
	Timestamp	time.Time `gorm:"column:timestamp;default:CURRENT_TIMESTAMP"`
}

type Credential struct {
	ID			uint
	Username	string
	Password	string
	Note		string
}

type Task struct {
	ID			uint
	Note		string
	Status		string
	DueTime		time.Time
	AssigneeID	uint
	Assignee	UserData
}