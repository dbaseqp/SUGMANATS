package main

import (
	"fmt"
	// "io"
	"log"
	// "os"	
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"sugmanats/models"
)

var (
	tomlConf   = &models.Config{}
	configPath = "config.conf"
)

func main() {
	// setup config
	models.ReadConfig(tomlConf, configPath)
	err := models.CheckConfig(tomlConf)
	if err != nil {
		log.Fatalln(errors.Wrap(err, "illegal config"))
	}

	// setup database
	db, err := gorm.Open(sqlite.Open(tomlConf.DBPath), &gorm.Config{})
	if err != nil {
		log.Fatalln("Failed to connect database!")
	}
	db.AutoMigrate(&models.Box{})

	// setup router
	router := gin.Default()
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

	// alright, let's begin
	log.Fatalln(router.Run(":" + fmt.Sprint(tomlConf.Port)))
}
