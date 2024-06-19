package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/noirbizarre/gonja"
	"plexcorp.tech/scriptable/models"
	"plexcorp.tech/scriptable/utils"
)

func (c *Controller) ChooseServerType(gctx *gin.Context) {
	c.Render("servers/build", gonja.Context{
		"serverTypes": models.GetServerTypes(),
		"title":       "Choose server template",
	}, gctx)

}

func (c *Controller) Servers(gctx *gin.Context) {
	page, err := strconv.Atoi(gctx.Query("page"))
	sessUser := c.GetSessionUser(gctx)
	view := gctx.Query("view")
	status := gctx.Query("status")
	if status == "" {
		status = "all"
	}
	if err != nil {
		page = 1
	}

	perPage, err := strconv.Atoi(gctx.Query("perPage"))
	if err != nil {
		perPage = 20
	}

	search := gctx.Query("search")
	servers := models.GetServersList(c.GetDB(gctx), page, perPage, search, status, sessUser.TeamId)
	searchQuery := ""

	if search != "" {
		searchQuery = "&search=" + searchQuery
	}

	vars := gonja.Context{
		"title":       "Servers",
		"view":        view,
		"status":      status,
		"servers":     servers,
		"nextPage":    page + 1,
		"prevPage":    page - 1,
		"searchQuery": searchQuery,
		"search":      search,
		"numServers":  len(servers),
		"highlight":   "servers",
	}

	c.Render("servers/list", vars, gctx)

}

// Should a server build fail - this allows for re-trying, scriptables will automatically
// pick up from the last failed step and try to continue on with the build.
func (c *Controller) RetryBuildServer(gctx *gin.Context) {
	retryBuild := gctx.FormValue("retryBuildServerId")
	sessUser := c.GetSessionUser(gctx)
	db := c.GetDB(gctx)
	updated := false

	if retryBuild != "" {
		sid, err := strconv.ParseInt(retryBuild, 10, 64)
		if !models.IsMyServer(db, sid, sessUser.TeamId) {
			gctx.Redirect(http.StatusFound, "/denied")
			return
		}

		if err == nil && sid != 0 {
			db.Exec("UPDATE servers set status='queued' where status <> 'success' and id=? and team_id=?", sid, sessUser.TeamId)
			updated = true
		}
	}

	if !updated {
		c.FlashError(gctx, "Sorry, failed to queue server rebuild. Please try again.")
	} else {
		c.FlashSuccess(gctx, "Successfully queued server rebuild.")
	}

	gctx.Redirect(http.StatusFound, "/servers")
}

func (c *Controller) CreateServer(gctx *gin.Context) {
	var countSshKeys int64
	sessUser := c.GetSessionUser(gctx)

	c.GetDB(gctx).Table("ssh_keys").Where("team_id=?", sessUser.TeamId).Count(&countSshKeys)
	if countSshKeys == 0 {
		c.Render("general/warning", gonja.Context{
			"title":      "SSH keys not found",
			"warningMsg": "Please add at least one SSH Key <a href=\"/sshkey/create\"> here</a> first before trying to build a server.",
		}, gctx)

		return
	}

	keys := []models.SshKey{}
	c.GetDB(gctx).Where("team_id=?", sessUser.TeamId).Find(&keys)
	serverType := gctx.Param("servertype")

	server := models.Server{ServerType: serverType}
	server.NewSSHUsername = "developer"
	server.SSHUsername = "root"
	server.SshPort = 22
	server.NewSshPort = 2022

	var errors []string

	if gctx.Request.Method == http.MethodPost {
		errors = models.ValidateForm(gctx, &server)
	}

	vars := gonja.Context{
		"serverTypes":       models.GetServerTypes(),
		"sshKeys":           keys,
		"ServerName":        server.ServerName,
		"ServerType":        server.ServerType,
		"ServerIP":          server.ServerIP,
		"PrivateServerIP":   server.PrivateServerIP,
		"SSHKeyId":          server.SSHKeyId,
		"SSHUsername":       server.SSHUsername,
		"NewSSHUsername":    server.NewSSHUsername,
		"Redis":             server.Redis,
		"Certbot":           server.Certbot,
		"Memcache":          server.Memcache,
		"MySql":             server.MySql,
		"MySqlRootPassword": server.MySqlRootPassword,
		"PhpVersion":        server.PhpVersion,
		"WebserverType":     server.WebserverType,
		"Status":            server.Status,
		"ScriptableName":    server.ScriptableName,
		"SshPort":           server.SshPort,
		"NewSshPort":        server.NewSshPort,
		"AptPackages":       server.AptPackages,
		"action":            "/server/create/" + server.ServerType,
		"actionType":        "BUILD SERVER",
		"title":             "Build new " + server.ServerType + " server",
		"highlight":         "servers",
	}

	if gctx.Request.Method == http.MethodPost && len(errors) > 0 {
		vars["errors"] = errors
	} else if gctx.Request.Method == http.MethodPost {
		server.Status = models.STATUS_CONNECTING
		server.UpdatedAt = time.Now()
		server.CreatedAt = time.Now()

		if server.MySqlRootPassword != "" {
			server.MySqlRootPassword = utils.Encrypt(server.MySqlRootPassword)
		}

		sessUser := c.GetSessionUser(gctx)
		server.TeamId = sessUser.TeamId

		c.GetDB(gctx).Create(&server)
		c.FlashSuccess(gctx, "Successfully saved. We are now testing the connection to this server...")
		gctx.Redirect(http.StatusFound, fmt.Sprintf("/server/test-ssh/%d", server.ID))
		return
	}

	c.Render("servers/form", vars, gctx)

}

func (c *Controller) UpdateServer(gctx *gin.Context) {
	keys := []models.SshKey{}
	sessUser := c.GetSessionUser(gctx)
	db := c.GetDB(gctx)
	db.Where("team_id=?", sessUser.TeamId).Find(&keys)

	serverId, _ := strconv.ParseInt(gctx.Param("id"), 10, 64)
	server := models.GetServerSimple(db, serverId, sessUser.TeamId)
	var errors []string

	if !models.IsMyServer(db, serverId, sessUser.TeamId) {
		gctx.Redirect(http.StatusFound, "/denied")
		return
	}

	if gctx.Request.Method == http.MethodPost {
		errors = models.ValidateForm(gctx, server)
	}

	vars := gonja.Context{
		"serverTypes":       models.GetServerTypes(),
		"sshKeys":           keys,
		"ServerName":        server.ServerName,
		"ServerType":        server.ServerType,
		"ServerIP":          server.ServerIP,
		"PrivateServerIP":   server.PrivateServerIP,
		"SSHKeyId":          server.SSHKeyId,
		"SSHUsername":       server.SSHUsername,
		"NewSSHUsername":    server.NewSSHUsername,
		"Redis":             server.Redis,
		"Certbot":           server.Certbot,
		"Memcache":          server.Memcache,
		"MySql":             server.MySql,
		"MySqlRootPassword": utils.Decrypt(server.MySqlRootPassword),
		"PhpVersion":        server.PhpVersion,
		"WebserverType":     server.WebserverType,
		"Status":            server.Status,
		"ScriptableName":    server.ScriptableName,
		"SshPort":           server.SshPort,
		"NewSshPort":        server.NewSshPort,
		"AptPackages":       server.AptPackages,
		"action":            fmt.Sprintf("/server/update/%d", server.ID),
		"actionType":        "UPDATE SERVER",
		"title":             "Update server: " + server.ServerName,
		"highlight":         "servers",
	}

	if gctx.Request.Method == http.MethodPost && len(errors) > 0 {
		vars["errors"] = errors
	} else if gctx.Request.Method == http.MethodPost {
		server.Status = models.STATUS_CONNECTING
		server.UpdatedAt = time.Now()
		server.CreatedAt = time.Now()

		if server.MySqlRootPassword != "" {
			server.MySqlRootPassword = utils.Encrypt(server.MySqlRootPassword)
		}
		c.GetDB(gctx).Save(&server)
		c.FlashSuccess(gctx, "Successfully updated. We are now testing the connection to this server...")
		gctx.Redirect(http.StatusFound, fmt.Sprintf("/server/test-ssh/%d", server.ID))
		return
	}

	c.Render("servers/form", vars, gctx)

}

// When you create a new server, Scriptables will automatically try and
// establish an SSH connection to your server using the linked SSH key pair.
// Should something go wrong, you'll be notified and can either change your SSH key or retry.
func (c *Controller) ShowTestConnectionLoader(gctx *gin.Context) {
	id, _ := strconv.ParseInt(gctx.Param("id"), 10, 64)
	sessUser := c.GetSessionUser(gctx)

	db := c.GetDB(gctx)
	server := models.GetServer(db, id, sessUser.TeamId)
	if !models.IsMyServer(db, id, sessUser.TeamId) {
		gctx.Redirect(http.StatusFound, "/denied")
		return
	}

	c.Render("servers/sshtest", gonja.Context{
		"title":      "Testing SSH connection to your server...",
		"id":         id,
		"serverName": server.ServerName,
		"serverIP":   server.ServerIP,
		"highlight":  "servers",
	}, gctx)
}

func (c *Controller) TestSSHConnection(gctx *gin.Context) {
	id, _ := strconv.ParseInt(gctx.Param("id"), 10, 64)
	sessUser := c.GetSessionUser(gctx)
	db := c.GetDB(gctx)
	server := models.GetServer(db, id, sessUser.TeamId)

	if !models.IsMyServer(db, id, sessUser.TeamId) {
		gctx.Redirect(http.StatusFound, "/denied")
		return
	}

	response := make(map[string]string)
	errorMsg := "Sorry, connection failed. Please check your server settings especially the SSH key - it should be the same key used when creating the server."

	response["id"] = fmt.Sprintf("%d", server.ID)
	response["status"] = "failed"
	response["message"] = errorMsg

	if server.ID == 0 {
		response["status"] = "failed"
		response["message"] = errorMsg
		gctx.JSON(http.StatusBadRequest, response)
		return
	}

	connection, err := models.GetSSHClient(&server, true)
	if err == nil {
		connection.Close()
		c.GetDB(gctx).Exec("UPDATE servers SET status=? WHERE id=?", models.STATUS_QUEUED, server.ID)
		response["status"] = "success"
		response["message"] = "Success! now attempting deploy. Please check server logs for progress and more information."
		gctx.JSON(http.StatusOK, response)
		return
	} else {
		c.GetDB(gctx).Exec("UPDATE servers set status=? WHERE id = ?", models.STATUS_FAILED, server.ID)
	}

	gctx.JSON(http.StatusBadRequest, response)

}

func (c *Controller) FirewallRules(gctx *gin.Context) {
	id, _ := strconv.ParseInt(gctx.Param("serverID"), 10, 64)
	sessUser := c.GetSessionUser(gctx)
	db := c.GetDB(gctx)

	if !models.IsMyServer(db, id, sessUser.TeamId) {
		gctx.Redirect(http.StatusFound, "/denied")
		return
	}

	server := models.GetServer(db, id, sessUser.TeamId)

	c.Render("servers/firewall_rules", gonja.Context{
		"title":      "Server firewall rules",
		"serverID":   id,
		"serverName": server.ServerName,
		"highlight":  "servers",
	}, gctx)
}

func (c *Controller) FirewallRulesAjax(gctx *gin.Context) {
	id, err := strconv.ParseInt(gctx.Param("serverID"), 10, 64)
	db := c.GetDB(gctx)
	sessUser := c.GetSessionUser(gctx)
	var server models.ServerWithSShKey

	if !models.IsMyServer(db, id, sessUser.TeamId) {
		gctx.Redirect(http.StatusFound, "/denied")
		return
	}

	response := make(map[string]string)
	if err != nil {
		response["status"] = "failed"
		response["msg"] = "Invalid server ID"
		gctx.JSON(http.StatusBadRequest, response)
		return
	}

	server = models.GetServer(db, id, sessUser.TeamId)
	if server.ID == 0 {
		response["status"] = "failed"
		response["msg"] = "Invalid server ID"
		gctx.JSON(http.StatusBadRequest, response)
		return
	}

	client, err := models.GetSSHClient(&server, false)

	if err != nil {
		response["status"] = "failed"
		response["msg"] = err.Error()
		gctx.JSON(http.StatusBadRequest, response)
		return
	}

	rules, err := models.GetRules(client)

	if err != nil {
		response["status"] = "failed"
		response["msg"] = err.Error()
		gctx.JSON(http.StatusBadRequest, response)
		return
	}

	gctx.JSON(http.StatusOK, rules)
}

func (c *Controller) DeleteFirewallRule(gctx *gin.Context) {
	serverID, _ := strconv.ParseInt(gctx.FormValue("server_id"), 10, 64)
	ruleNumber, _ := strconv.ParseInt(gctx.FormValue("rule_number"), 10, 64)
	sessUser := c.GetSessionUser(gctx)
	rule := gctx.FormValue("rule")

	db := c.GetDB(gctx)

	if !models.IsMyServer(db, serverID, sessUser.TeamId) {
		gctx.Redirect(http.StatusFound, "/denied")
		return
	}

	response := make(map[string]string)

	if serverID == 0 || ruleNumber == 0 {
		response["status"] = "failed"
		response["msg"] = "Bad server or rule ID."
		gctx.JSON(http.StatusBadRequest, response)
		return
	}

	server := models.GetServer(db, serverID, sessUser.TeamId)
	if server.ID == 0 {
		response["status"] = "failed"
		response["msg"] = "Bad server or rule ID."
		gctx.JSON(http.StatusBadRequest, response)
		return
	}

	err := models.DeleteFirewallRule(db, &server, ruleNumber, rule)

	if err != nil {
		response["status"] = "failed"
		response["msg"] = "Bad server or rule ID."
		gctx.JSON(http.StatusBadRequest, response)
		return
	}

	response["status"] = "success"
	response["msg"] = "Successfully deleted rule."
	gctx.JSON(http.StatusOK, response)
}

func (c *Controller) AddFirewallRule(gctx *gin.Context) {
	serverID, _ := strconv.ParseInt(gctx.FormValue("server_id"), 10, 64)
	sessUser := c.GetSessionUser(gctx)
	rule := gctx.FormValue("rule")
	db := c.GetDB(gctx)

	if !models.IsMyServer(db, serverID, sessUser.TeamId) {
		gctx.Redirect(http.StatusFound, "/denied")
		return
	}

	response := make(map[string]string)

	if serverID == 0 || rule == "" {
		response["status"] = "failed"
		response["msg"] = "Bad server or firewall rule."
		gctx.JSON(http.StatusBadRequest, response)
		return
	}

	server := models.GetServer(db, serverID, sessUser.TeamId)
	if server.ID == 0 {
		response["status"] = "failed"
		response["msg"] = "Bad server ID."
		gctx.JSON(http.StatusBadRequest, response)
		return
	}

	rule = strings.ToLower(rule)

	if strings.Contains(rule, "anywhere") {
		rule = strings.ReplaceAll(rule, "to anywhere port", "")
		rule = strings.ReplaceAll(rule, "from anywhere to any port", "")
		rule = strings.ReplaceAll(rule, "  ", " ")
		rule = strings.ReplaceAll(rule, " proto tcp", "/tcp")
		rule = strings.ReplaceAll(rule, " proto udp", "/udp")
	}

	rule = strings.ReplaceAll(rule, "port any proto", "proto")

	err := models.AddFirewallRule(db, &server, rule)

	if err != nil {
		response["status"] = "failed"
		response["msg"] = "Failed to add firewall rule. Please try again."
		gctx.JSON(http.StatusBadRequest, response)
		return
	}

	response["status"] = "success"
	response["msg"] = "Successfully added firewall rule."
	gctx.JSON(http.StatusOK, response)
}
