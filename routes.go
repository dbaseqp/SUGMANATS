package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func addPublicRoutes(g *gin.RouterGroup) {
	g.GET("/", viewIndex)
	g.GET("/login", viewLogin)
	g.POST("/login", login)
}

func addPrivateRoutes(g *gin.RouterGroup) {
	g.GET("/logout", logout)
	g.GET("/dashboard", viewDashboard)
}

func pageData(c *gin.Context, title string, ginMap gin.H) gin.H {
	newGinMap := gin.H{}
	newGinMap["title"] = title
	newGinMap["user"] = getUser(c)
	newGinMap["config"] = tomlConf
	newGinMap["operation"] = tomlConf.Operation
	for key, value := range ginMap {
		newGinMap[key] = value
	}
	return newGinMap
}

func viewIndex (c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", pageData(c, "SUGMANATS", nil))
}

func viewLogin (c *gin.Context) {
	if getUser(c).IsValid() {
		c.Redirect(http.StatusSeeOther, "/dashboard")
	}
	c.HTML(http.StatusOK, "login.html", pageData(c, "Login", nil))
}

func viewDashboard (c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard.html", pageData(c, "Dashboard", nil))
}