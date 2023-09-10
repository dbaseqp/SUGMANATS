package main

import (
	"fmt"
	// "io"
	"log"
	// "os"	
	"net/http"
	"html/template"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"sugmanats/models"
)

var (
	tomlConf   	= &models.Config{}
	configPath 	= "config.conf"
	db			= &gorm.DB{}
	stream		= NewSSEServer()
)

type ClientChan chan string

type Event struct {
	// Events are pushed to this channel by the main events-gathering routine
	Message chan string

	// New client connections
	NewClients chan chan string

	// Closed client connections
	ClosedClients chan chan string

	// Total client connections
	TotalClients map[chan string]bool
}

func main() {
	// setup config
	models.ReadConfig(tomlConf, configPath)
	err := models.CheckConfig(tomlConf)
	if err != nil {
		log.Fatalln(errors.Wrap(err, "illegal config"))
	}

	// setup database
	db, err = gorm.Open(sqlite.Open(tomlConf.DBPath), &gorm.Config{})
	if err != nil {
		log.Fatalln("Failed to connect database!")
	}
	db.AutoMigrate(&models.Box{}, &models.Port{}, &models.UserData{}, &models.Credential{})

	dbAddUsers(tomlConf.Admin)

	// setup router
	router := gin.Default()
	router.SetFuncMap(template.FuncMap{
        "markdown": models.RenderMarkdown,
    })
	router.Static("/assets", "./assets")
	router.LoadHTMLGlob("templates/*.html")
	router.MaxMultipartMemory = 8 << 20 // 8Mib
	initCookies(router)


	// setup routes
	router.NoRoute(func(c *gin.Context) {
		c.HTML(http.StatusNotFound, "404.html", pageData(c, "404 Not Found", nil))
	})
	public := router.Group("/")
	addPublicRoutes(public)

	private := router.Group("/")
	private.Use(authRequired)
	addPrivateRoutes(private)

	// Create a channel to send SSE messages
	// messageChannel = make(chan string)

	// Close the channel when the client connection closes
	// defer func() {
	// 	log.Println("Closing SSE...")
	// 	close(messageChannel)
	// }()

	// alright, let's begin
	log.Fatalln(router.Run(":" + fmt.Sprint(tomlConf.Port)))
}

func NewSSEServer() (event *Event) {
	event = &Event{
		Message:       make(chan string),
		NewClients:    make(chan chan string),
		ClosedClients: make(chan chan string),
		TotalClients:  make(map[chan string]bool),
	}

	go event.listen()

	return
}

func (stream *Event) listen() {
	for {
		select {
		// Add new available client
		case client := <-stream.NewClients:
			stream.TotalClients[client] = true
			log.Printf("Client added. %d registered clients", len(stream.TotalClients))

		// Remove closed client
		case client := <-stream.ClosedClients:
			delete(stream.TotalClients, client)
			close(client)
			log.Printf("Removed client. %d registered clients", len(stream.TotalClients))

		// Broadcast message to client
		case eventMsg := <-stream.Message:
			for clientMessageChan := range stream.TotalClients {
				clientMessageChan <- eventMsg
			}
		}
	}
}

func (stream *Event) serveHTTP() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Initialize client channel
		clientChan := make(ClientChan)

		// Send new connection to event server
		stream.NewClients <- clientChan

		defer func() {
			// Send closed connection to event server
			stream.ClosedClients <- clientChan
		}()

		c.Set("clientChan", clientChan)

		c.Next()
	}
}