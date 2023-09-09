package main

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"sugmanats/models"
)

// authRequired provides authentication middleware for ensuring that a user is logged in.
func authRequired(c *gin.Context) {
	session := sessions.Default(c)
	id := session.Get("id")
	if id == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		c.Abort()
	}
	c.Next()
}

// getUUID returns a randomly generated UUID
func getUUID() string {
	return uuid.New().String()
}

// initCookies use gin-contrib/sessions{/cookie} to initalize a cookie store.
// It generates a random secret for the cookie store -- not ideal for continuity or invalidating previous cookies, but it's secure and it works
func initCookies(router *gin.Engine) {
	//r.Use(sessions.Sessions("dwayne-inator-5000", cookie.NewStore([]byte(getUUID()))))
	router.Use(sessions.Sessions("amungos2", cookie.NewStore([]byte("sooper secure"))))
}

// login is a handler that parses a form and checks for specific data
func login(c *gin.Context) {
	session := sessions.Default(c)
	username := c.PostForm("username")
	password := c.PostForm("password")
	var user models.UserData

	// Validate form input
	if strings.Trim(username, " ") == "" || strings.Trim(password, " ") == "" {
		c.HTML(http.StatusBadRequest, "login.html", pageData(c, "login", gin.H{"error": "Username or password can't be empty."}))
		return
	}

	err := errors.New("Invalid username or password.")

	for i, t := range tomlConf.Admin {
		if username == t.Name && password == t.Pw {
			user = t
			user.ID = uint(i + 1)
			err = nil
		}
	}

	if err != nil {
		c.HTML(http.StatusBadRequest, "login.html", pageData(c, "login", gin.H{"error": err.Error()}))
		return
	}

	// Save the username in the session
	session.Set("id", user.ID)
	if err := session.Save(); err != nil {
		c.HTML(http.StatusBadRequest, "login.html", pageData(c, "login", gin.H{"error": "Failed to save session."}))
		return
	}
	c.Redirect(http.StatusSeeOther, "/dashboard")
}

func getUser(c *gin.Context) models.UserData {
	userID := sessions.Default(c).Get("id")
	if userID != nil {
		dbUser, err := dbGetUser(userID.(uint))
		if err != nil {
			return models.UserData{}
		}
		return dbUser
	}
	return models.UserData{}
}

func logout(c *gin.Context) {
	session := sessions.Default(c)
	id := session.Get("id")
	if id == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	session.Delete("id")
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
		return
	}
	c.Redirect(http.StatusSeeOther, "/")
}
