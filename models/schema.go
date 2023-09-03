package models

type Box struct {
	ID   uint
	Name string
	IP   string
	Port int
}

type UserData struct {
	ID          uint
	Name		string
	Pw			string
}