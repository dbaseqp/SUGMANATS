package main

import (
	"net/http"
	"bytes"
	"mime/multipart"
	"log"
	"fmt"
	"io"
	"encoding/xml"
	"encoding/json"
	"strconv"
	"time"

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
	// generic
	g.GET("/about", viewAbout)
	g.GET("/logout", logout)
	g.GET("/dashboard", viewDashboard)
	g.GET("/settings", viewSettings)
	g.POST("/settings/edit/:userId", editSettings)
	g.GET("/sse", stream.serveHTTP(), sse)
	
	/* inventory */
	// boxes
	g.GET("/boxes", viewBoxes)
	g.GET("/boxes/export", viewExportBoxes)
	g.POST("/boxes/upload", uploadNmap)
	g.POST("/boxes/edit/details/:boxId", editBoxDetails)
	g.POST("/boxes/edit/note/:boxId", editBoxNote)
	g.GET("/api/boxes", getBoxes)

	// credentials
	g.GET("/credentials", viewCredentials)
	g.POST("/credentials/add", addCredential)
	g.POST("/credentials/edit/:credentialId", editCredential)
	g.POST("/credentials/delete/:credentialId", deleteCredential)

	/* tasks */
	g.GET("/tasks", viewTasks)
	g.POST("/tasks/add", addTask)
	g.POST("/tasks/edit/:taskId", editTask)
	g.POST("/tasks/delete/:taskId", deleteTask)
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
		c.HTML(http.StatusInternalServerError, "dashboard.html", pageData(c, "Export Boxes", gin.H{"error": err}))
		return
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

func viewSettings (c *gin.Context) {
	user := getUser(c)
	c.HTML(http.StatusOK, "settings.html", pageData(c, "Settings", gin.H{"user": user}))
}

func editSettings (c *gin.Context) {
	user := getUser(c)
	user.Color 	= c.PostForm("color")
	err := dbEditSettings(&user)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": errors.Wrap(err, "Error").Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": true, "message":"Saved changes!"})
}

func viewBoxes (c *gin.Context) {
	boxes, err := dbGetBoxes()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "export-boxes.html", pageData(c, "Export Boxes", gin.H{"error": err}))
		return
	}
	ports, err := dbGetPorts()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "export-boxes.html", pageData(c, "Export Boxes", gin.H{"error": err}))
		return
	}
	users, err := dbGetUsers()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "export-boxes.html", pageData(c, "Export Boxes", gin.H{"error": err}))
		return
	}
	c.HTML(http.StatusOK, "boxes.html", pageData(c, "Boxes", gin.H{"boxes": boxes, "ports": ports, "users": users}))
}

func viewExportBoxes (c *gin.Context) {
	boxes, err := dbGetBoxes()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "export-boxes.html", pageData(c, "Export Boxes", gin.H{"error": err}))
		return
	}
	ports, err := dbGetPorts()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "export-boxes.html", pageData(c, "Export Boxes", gin.H{"error": err}))
		return
	}
	users, err := dbGetUsers()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "export-boxes.html", pageData(c, "Export Boxes", gin.H{"error": err}))
		return
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
	var dataErrors []string
	var fileErrors int
	var boxCount int
	 
	for _, formFile := range form.Files {
		errorOnIteration := false
		openedFile, _ := formFile.Open()
		file, _ := io.ReadAll(openedFile)

		xml.Unmarshal(file, &nmapXML)
		log.Println(fmt.Sprintf("Upload %s success!", formFile.Filename))
		for _, host := range nmapXML.Host {
			boxCount++

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
			box, err = dbPropagateData(&box)
			if err != nil {  // tbh idk if this even needs error checking, but who knows...
				dataErrors = append(dataErrors, errors.Wrap(err, "Data propagation error:").Error())
				errorOnIteration = true
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
		if (errorOnIteration) {
			fileErrors++
		}
	}
	if len(dataErrors) != 0 {
		for err := range dataErrors {
			c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": err})
		}
	}
	sendSSE([]string{"boxes","ports","dirty"})
	c.JSON(http.StatusOK, gin.H{"status": true, "message":fmt.Sprintf("Received %d file(s) successfully! Found %d box(es) successfully.", len(form.Files) - fileErrors, boxCount - len(dataErrors))})
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
	sendSSE([]string{"boxes"})
	c.JSON(http.StatusOK, gin.H{"status": true, "message":"Updated box details successfully!"})
}

func editBoxNote (c *gin.Context) {
	boxId, err		:= strconv.ParseUint(c.Param("boxId"), 10, 32)
	note 			:= c.PostForm("note")

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": errors.Wrap(err, "Error").Error()})
		return
	}

	updatedBox := models.Box{
		ID: 		uint(boxId),
		Note: 		note,
	}

	err = dbUpdateBoxNote(&updatedBox)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": errors.Wrap(err, "Error").Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": true, "message":"Updated box note successfully!"})
}

func viewCredentials (c *gin.Context) {
	boxes, err := dbGetBoxes()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "export-boxes.html", pageData(c, "Export Boxes", gin.H{"error": err}))
		return
	}
	ports, err := dbGetPorts()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "export-boxes.html", pageData(c, "Export Boxes", gin.H{"error": err}))
		return
	}
	credentials, err := dbGetCredentials()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "credentials.html", pageData(c, "Credentials", gin.H{"error": err}))
		return
	}
	c.HTML(http.StatusOK, "credentials.html", pageData(c, "Credentials", gin.H{"boxes": boxes, "ports": ports, "credentials": credentials}))
}

func addCredential (c *gin.Context) {
	username 		:= c.PostForm("username")
	password 		:= c.PostForm("password")
	note	 		:= c.PostForm("note")

	newCredential := models.Credential{
		Username:	username,
		Password:	password,
		Note: 		note,
	}

	err := dbAddCredential(&newCredential)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": errors.Wrap(err, "Error").Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": true, "message":"Added credential successfully!"})
}

func editCredential (c *gin.Context) {
	credentialId, err		:= strconv.ParseUint(c.Param("credentialId"), 10, 32)
	username 				:= c.PostForm("username")
	password 				:= c.PostForm("password")
	note	 				:= c.PostForm("note")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": errors.Wrap(err, "Error").Error()})
		return
	}

	updatedCredential := models.Credential{
		ID:			uint(credentialId),
		Username:	username,
		Password:	password,
		Note: 		note,
	}

	err = dbEditCredential(&updatedCredential)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": errors.Wrap(err, "Error").Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": true, "message":"Edited credential successfully!"})
}

func deleteCredential (c *gin.Context) {
	credentialId, err		:= strconv.ParseUint(c.Param("credentialId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": errors.Wrap(err, "Error").Error()})
		return
	}

	err = dbDeleteCredential(uint(credentialId))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": errors.Wrap(err, "Error").Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": true, "message":"Deleted credential successfully!"})
}

func sse (c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Transfer-Encoding", "chunked")

	// Continuously send messages to the client
	v, ok := c.Get("clientChan")
	if !ok {
		return
	}
	clientChan, ok := v.(ClientChan)
	if !ok {
		return
	}
	c.Stream(func(w io.Writer) bool {
		// Stream message to client from message channel
		if msg, ok := <-clientChan; ok {
			c.SSEvent("message", msg)
			return true
		}
		return false
	})
}

func sendSSE(models []string) {
	if stream != nil {
		jsonString, err := json.Marshal(models)
		if err != nil {
			log.Printf("%s: %+v", errors.Wrap(err, "SSE json error").Error(), models)
			return
		}
		// send the message through the available channel
		stream.Message <- string(jsonString)
	}
}

func getBoxes(c *gin.Context) {
	boxes, err := dbGetBoxes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": errors.Wrap(err, "AJAX").Error()})
		return
	}
	users, err := dbGetUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": errors.Wrap(err, "AJAX").Error()})
		return
	}
	var boxIds []int
	jsonBoxes := map[int]models.Box{}
	for _, box := range boxes {
		boxIds = append(boxIds, int(box.ID))
		jsonBoxes[int(box.ID)] = box
	}

	c.JSON(http.StatusOK, gin.H{"status": true, "boxIds": boxIds, "boxes": jsonBoxes, "users": users})
}

func viewTasks (c *gin.Context) {
	users, err := dbGetUsers()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "tasks.html", pageData(c, "Tasks", gin.H{"error": err}))
		return
	}
	tasks, err := dbGetTasks()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "tasks.html", pageData(c, "Tasks", gin.H{"error": err}))
		return
	}
	c.HTML(http.StatusOK, "tasks.html", pageData(c, "Tasks", gin.H{"users": users, "tasks": tasks}))
}

type TaskForm struct {
    Assignee 	*int `form:"assignee" binding:"required"`
	Note		string `form:"note" binding:"required"`
	Status		string `form:"status" binding:"required"`
	DueTime		time.Time `form:"due-time"`
}

func addTask (c *gin.Context) {
	var form TaskForm
	err := c.ShouldBind(&form)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": errors.Wrap(err, "Error").Error()})
		return
	}

	newTask := models.Task{
		AssigneeID: uint(*form.Assignee),
		Status:		form.Status,
		Note: 		form.Note,
	}
	if !form.DueTime.IsZero() {
		newTask.DueTime = form.DueTime
	}

	err = dbAddTask(&newTask)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": errors.Wrap(err, "Error").Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": true, "message":"Added task successfully!"})
}

func editTask (c *gin.Context) {
	var form TaskForm
	err := c.ShouldBind(&form)
	taskId, err	:= strconv.ParseUint(c.Param("taskId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": errors.Wrap(err, "Error").Error()})
		return
	}

	newTask := models.Task{
		ID:			uint(taskId),
		DueTime:	form.DueTime,
		Status:		form.Status,
		Note: 		form.Note,
		AssigneeID:	uint(*form.Assignee),
	}

	err = dbEditTask(&newTask)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": errors.Wrap(err, "Error").Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": true, "message":"Edited task successfully!"})
}

func deleteTask (c *gin.Context) {
	taskId, err		:= strconv.ParseUint(c.Param("taskId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": errors.Wrap(err, "Error").Error()})
		return
	}

	err = dbDeleteTask(uint(taskId))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": errors.Wrap(err, "Error").Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": true, "message":"Deleted task successfully!"})
}