package main

import (
	"net/http"
	"bytes"
	"mime/multipart"
	"log"
	"fmt"
	"io"
	"encoding/xml"
	"strconv"

	"github.com/pkg/errors"
	"github.com/BurntSushi/toml"
	"github.com/gin-gonic/gin"

	"sugmanats/models"
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
	g.GET("/boxes/export", viewExportBoxes)
	g.POST("/boxes/upload", uploadNmap)
	g.POST("/boxes/edit/:boxId", editBoxDetails)
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
	boxes, err := dbGetBoxes()
	if err != nil {
		c.HTML(http.StatusOK, "dashboard.html", pageData(c, "Export Boxes", gin.H{"error": err}))
	}

	pwnCount 	:= 0
	usershells 	:= 0
	rootshells 	:= 0
	for _, box := range boxes {
		if box.Usershells > 0 || box.Rootshells > 0 {
			pwnCount++
			usershells += box.Usershells
			rootshells += box.Rootshells
		}
	}
	c.HTML(http.StatusOK, "dashboard.html", pageData(c, "Dashboard", gin.H{"boxes": boxes, "pwnCount": pwnCount, "percent": (100*float32(pwnCount)/float32(len(boxes))), "usershells": usershells, "rootshells": rootshells}))
}

func viewBoxes (c *gin.Context) {
	boxes, err := dbGetBoxes()
	if err != nil {
		c.HTML(http.StatusOK, "export-boxes.html", pageData(c, "Export Boxes", gin.H{"error": err}))
	}
	ports, err := dbGetPorts()
	if err != nil {
		c.HTML(http.StatusOK, "export-boxes.html", pageData(c, "Export Boxes", gin.H{"error": err}))
	}
	users, err := dbGetUsers()
	if err != nil {
		c.HTML(http.StatusOK, "export-boxes.html", pageData(c, "Export Boxes", gin.H{"error": err}))
	}
	c.HTML(http.StatusOK, "boxes.html", pageData(c, "Boxes", gin.H{"boxes": boxes, "ports": ports, "users": users}))
}

func viewExportBoxes (c *gin.Context) {
	boxes, err := dbGetBoxes()
	if err != nil {
		c.HTML(http.StatusOK, "export-boxes.html", pageData(c, "Export Boxes", gin.H{"error": err}))
	}
	ports, err := dbGetPorts()
	if err != nil {
		c.HTML(http.StatusOK, "export-boxes.html", pageData(c, "Export Boxes", gin.H{"error": err}))
	}
	users, err := dbGetUsers()
	if err != nil {
		c.HTML(http.StatusOK, "export-boxes.html", pageData(c, "Export Boxes", gin.H{"error": err}))
	}
	c.HTML(http.StatusOK, "export-boxes.html", pageData(c, "Export Boxes", gin.H{"boxes": boxes, "ports": ports, "users": users}))
}

func viewAbout (c *gin.Context) {
	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(tomlConf); err != nil {
		c.HTML(http.StatusInternalServerError, "settings.html", pageData(c, "Settings", gin.H{"error": err}))
		return
	}
	c.HTML(http.StatusOK, "about.html", pageData(c, "About", gin.H{"config": buf.String()}))
}

type NmapUpload struct {
    Files []*multipart.FileHeader `form:"files" binding:"required"`
}

func uploadNmap (c *gin.Context) {
	var form NmapUpload
	err := c.ShouldBind(&form)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": errors.Wrap(err, "Error").Error()})
		return
	}

	var nmapXML models.Nmaprun
	var box models.Box
	var port models.Port
	for _, formFile := range form.Files {
		openedFile, _ := formFile.Open()
		file, _ := io.ReadAll(openedFile)

		xml.Unmarshal(file, &nmapXML)
		log.Println(fmt.Sprintf("Upload %s success!", formFile.Filename))
		for _, host := range nmapXML.Host {
			box = models.Box {
				Status: host.Status.State,
			}
			
			for _, address := range host.Address {
				if address.Addrtype == "ipv4" {
					box.IP = address.Addr
				}
			}

			hostname := ""
			for _, h := range host.Hostnames.Hostname {
				hostname = fmt.Sprintf("%s,%s/%s", hostname, h.Name, h.Type)
			}
			if len(hostname) > 2 {
				box.Hostname = hostname[1:]
			}
			_ = db.Create(&box)

			for _, p := range host.Ports.Port {
				port = models.Port{
					Port: p.Portid,
					BoxID: box.ID,
					Protocol: p.Protocol,
					State: p.State.State,
					Service: p.Service.Name,
					Tunnel: p.Service.Tunnel,
					Version: p.Service.Version,
				}
				db.Create(&port)
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"status": true, "message":"Received files successfully!"})
}

func editBoxDetails (c *gin.Context) {
	boxId, err		:= strconv.ParseUint(c.Param("boxId"), 10, 32)
	claimerId, err	:= strconv.ParseUint(c.PostForm("claim"), 10, 32)
	usershells, err	:= strconv.Atoi(c.PostForm("usershells"))
	rootshells, err	:= strconv.Atoi(c.PostForm("rootshells"))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": errors.Wrap(err, "Error").Error()})
		return
	}

	updatedBox := models.Box{
		ID: 		uint(boxId),
		Usershells: usershells,
		Rootshells: rootshells,
		ClaimerID:	uint(claimerId),
	}

	err = dbUpdateBoxDetails(&updatedBox)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": errors.Wrap(err, "Error").Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": true, "message":"Updated box details successfully!"})
}