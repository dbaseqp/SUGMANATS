package main

import (
	"net/http"
	"bytes"
	"github.com/BurntSushi/toml"

	"github.com/gin-gonic/gin"
)

func addPublicRoutes(g *gin.RouterGroup) {
	g.GET("/", viewIndex)
	g.GET("/login", viewLogin)
	g.POST("/login", login)
}

func addPrivateRoutes(g *gin.RouterGroup) {
	g.GET("/about", viewAbout)
	g.GET("/logout", logout)
	g.GET("/dashboard", viewDashboard)
	g.GET("/boxes", viewBoxes)
}

func pageData(c *gin.Context, title string, ginMap gin.H) gin.H {
	newGinMap := gin.H{}
	newGinMap["title"] = title
	newGinMap["user"] = getUser(c)
	newGinMap["config"] = tomlConf
	newGinMap["operation"] = tomlConf.Operation
	// newGinMap["boxes"] = boxes
	for key, value := range ginMap {
		newGinMap[key] = value
	}
	return newGinMap
}

// public routes

func viewIndex (c *gin.Context) {
	if !getUser(c).IsValid() {
		c.Redirect(http.StatusSeeOther, "/login")
	}
	c.HTML(http.StatusOK, "index.html", pageData(c, "SUGMANATS", nil))
}

func viewLogin (c *gin.Context) {
	if getUser(c).IsValid() {
		c.Redirect(http.StatusSeeOther, "/dashboard")
	}
	c.HTML(http.StatusOK, "login.html", pageData(c, "Login", nil))
}

// private routes

func viewDashboard (c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard.html", pageData(c, "Dashboard", nil))
}

func viewBoxes (c *gin.Context) {
	c.HTML(http.StatusOK, "boxes.html", pageData(c, "Boxes", nil))
}

func viewAbout (c *gin.Context) {
	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(tomlConf); err != nil {
		c.HTML(http.StatusInternalServerError, "settings.html", pageData(c, "Settings", gin.H{"error": err}))
		return
	}
	c.HTML(http.StatusOK, "about.html", pageData(c, "About", gin.H{"config": buf.String()}))
}