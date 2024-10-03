package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/noirbizarre/gonja"
	"plexcorp.tech/scriptable/models"
	"plexcorp.tech/scriptable/utils"
)

func (c *Controller) ChooseServerType(gctx echo.Context) error {
	return c.Render("servers/build", gonja.Context{
		"serverTypes": models.GetServerTypes(),
		"title":       "Choose server template",
	}, gctx)

}

func (c *Controller) Servers(gctx echo.Context) error {
	page, err := strconv.Atoi(gctx.QueryParam("page"))
	sessUser := c.GetSessionUser(gctx)
	view := gctx.QueryParam("view")
	status := gctx.QueryParam("status")
	if status == "" {
		status = "all"
	}
	if err != nil {
		page = 1
	}

	perPage, err := strconv.Atoi(gctx.QueryParam("perPage"))
	if err != nil {
		perPage = 20
	}

	search := gctx.QueryParam("search")
	servers := models.GetServersList(page, perPage, search, status, sessUser.TeamId)
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

	return c.Render("servers/list", vars, gctx)

}

// Should a server build fail - this allows for re-trying, scriptables will automatically
// pick up from the last failed step and try to continue on with the build.
func (c *Controller) RetryBuildServer(gctx echo.Context) error {
	retryBuild := gctx.FormValue("retryBuildServerId")
	sessUser := c.GetSessionUser(gctx)
	db := models.GetDB()
	updated := false

	if retryBuild != "" {
		sid, err := strconv.ParseInt(retryBuild, 10, 64)
		if !models.IsMyServer(sid, sessUser.TeamId) {
			return gctx.Redirect(http.StatusFound, "/denied")
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

	return gctx.Redirect(http.StatusFound, "/servers")
}

func (c *Controller) CreateServer(gctx echo.Context) error {
	var countSshKeys int64
	sessUser := c.GetSessionUser(gctx)
	db := models.GetDB()

	db.Table("ssh_keys").Where("team_id=?", sessUser.TeamId).Count(&countSshKeys)
	if countSshKeys == 0 {
		return c.Render("general/warning", gonja.Context{
			"title":      "SSH keys not found",
			"warningMsg": "Please add at least one SSH Key <a href=\"/sshkey/create\"> here</a> first before trying to build a server.",
		}, gctx)

	}

	keys := []models.SshKey{}
	db.Where("team_id=?", sessUser.TeamId).Find(&keys)
	serverType := gctx.Param("servertype")

	server := models.Server{ServerType: serverType}
	server.NewSSHUsername = "developer"
	server.SSHUsername = "root"
	server.SshPort = 22
	server.NewSshPort = 2022

	var errors []string

	if gctx.Request().Method == http.MethodPost {
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

	if gctx.Request().Method == http.MethodPost && len(errors) > 0 {
		vars["errors"] = errors
	} else if gctx.Request().Method == http.MethodPost {
		server.Status = models.STATUS_CONNECTING
		server.UpdatedAt = time.Now()
		server.CreatedAt = time.Now()

		if server.MySqlRootPassword != "" {
			server.MySqlRootPassword = utils.Encrypt(server.MySqlRootPassword)
		}

		sessUser := c.GetSessionUser(gctx)
		server.TeamId = sessUser.TeamId

		db.Create(&server)
		c.FlashSuccess(gctx, "Successfully saved. We are now testing the connection to this server...")
		return gctx.Redirect(http.StatusFound, fmt.Sprintf("/server/test-ssh/%d", server.ID))
	}

	return c.Render("servers/form", vars, gctx)

}

func (c *Controller) UpdateServer(gctx echo.Context) error {
	keys := []models.SshKey{}
	sessUser := c.GetSessionUser(gctx)
	db := models.GetDB()
	db.Where("team_id=?", sessUser.TeamId).Find(&keys)

	serverId, _ := strconv.ParseInt(gctx.Param("id"), 10, 64)
	server := models.GetServerSimple(serverId, sessUser.TeamId)
	var errors []string

	if !models.IsMyServer(serverId, sessUser.TeamId) {
		return gctx.Redirect(http.StatusFound, "/denied")

	}

	if gctx.Request().Method == http.MethodPost {
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

	if gctx.Request().Method == http.MethodPost && len(errors) > 0 {
		vars["errors"] = errors
	} else if gctx.Request().Method == http.MethodPost {
		server.Status = models.STATUS_CONNECTING
		server.UpdatedAt = time.Now()
		server.CreatedAt = time.Now()

		if server.MySqlRootPassword != "" {
			server.MySqlRootPassword = utils.Encrypt(server.MySqlRootPassword)
		}
		db.Save(&server)
		c.FlashSuccess(gctx, "Successfully updated. We are now testing the connection to this server...")
		return gctx.Redirect(http.StatusFound, fmt.Sprintf("/server/test-ssh/%d", server.ID))
	}

	return c.Render("servers/form", vars, gctx)

}

// When you create a new server, Scriptables will automatically try and
// establish an SSH connection to your server using the linked SSH key pair.
// Should something go wrong, you'll be notified and can either change your SSH key or retry.
func (c *Controller) ShowTestConnectionLoader(gctx echo.Context) error {
	id, _ := strconv.ParseInt(gctx.Param("id"), 10, 64)
	sessUser := c.GetSessionUser(gctx)
	server := models.GetServer(id, sessUser.TeamId)
	if !models.IsMyServer(id, sessUser.TeamId) {
		return gctx.Redirect(http.StatusFound, "/denied")
	}

	return c.Render("servers/sshtest", gonja.Context{
		"title":      "Testing SSH connection to your server...",
		"id":         id,
		"serverName": server.ServerName,
		"serverIP":   server.ServerIP,
		"highlight":  "servers",
	}, gctx)
}

func (c *Controller) TestSSHConnection(gctx echo.Context) error {
	id, _ := strconv.ParseInt(gctx.Param("id"), 10, 64)
	sessUser := c.GetSessionUser(gctx)
	server := models.GetServer(id, sessUser.TeamId)

	if !models.IsMyServer(id, sessUser.TeamId) {
		return gctx.Redirect(http.StatusFound, "/denied")
	}

	response := make(map[string]string)
	errorMsg := "Sorry, connection failed. Please check your server settings especially the SSH key - it should be the same key used when creating the server."

	response["id"] = fmt.Sprintf("%d", server.ID)
	response["status"] = "failed"
	response["message"] = errorMsg

	if server.ID == 0 {
		response["status"] = "failed"
		response["message"] = errorMsg
		return gctx.JSON(http.StatusBadRequest, response)
	}

	connection, err := models.GetSSHClient(&server, true)
	db := models.GetDB()
	if err == nil {
		connection.Close()
		db.Exec("UPDATE servers SET status=? WHERE id=?", models.STATUS_QUEUED, server.ID)
		response["status"] = "success"
		response["message"] = "Success! now attempting deploy. Please check server logs for progress and more information."
		return gctx.JSON(http.StatusOK, response)
	} else {
		db.Exec("UPDATE servers set status=? WHERE id = ?", models.STATUS_FAILED, server.ID)
	}

	return gctx.JSON(http.StatusBadRequest, response)

}

func (c *Controller) FirewallRules(gctx echo.Context) error {
	id, _ := strconv.ParseInt(gctx.Param("serverID"), 10, 64)
	sessUser := c.GetSessionUser(gctx)

	if !models.IsMyServer(id, sessUser.TeamId) {
		return gctx.Redirect(http.StatusFound, "/denied")
	}

	server := models.GetServer(id, sessUser.TeamId)

	return c.Render("servers/firewall_rules", gonja.Context{
		"title":      "Server firewall rules",
		"serverID":   id,
		"serverName": server.ServerName,
		"highlight":  "servers",
	}, gctx)
}

func (c *Controller) FirewallRulesAjax(gctx echo.Context) error {
	id, err := strconv.ParseInt(gctx.Param("serverID"), 10, 64)
	sessUser := c.GetSessionUser(gctx)
	var server models.ServerWithSShKey

	if !models.IsMyServer(id, sessUser.TeamId) {
		return gctx.Redirect(http.StatusFound, "/denied")
	}

	response := make(map[string]string)
	if err != nil {
		response["status"] = "failed"
		response["msg"] = "Invalid server ID"
		return gctx.JSON(http.StatusBadRequest, response)
	}

	server = models.GetServer(id, sessUser.TeamId)
	if server.ID == 0 {
		response["status"] = "failed"
		response["msg"] = "Invalid server ID"
		return gctx.JSON(http.StatusBadRequest, response)
	}

	client, err := models.GetSSHClient(&server, false)

	if err != nil {
		response["status"] = "failed"
		response["msg"] = err.Error()
		return gctx.JSON(http.StatusBadRequest, response)
	}

	rules, err := models.GetRules(client)

	if err != nil {
		response["status"] = "failed"
		response["msg"] = err.Error()
		return gctx.JSON(http.StatusBadRequest, response)
	}

	return gctx.JSON(http.StatusOK, rules)
}

func (c *Controller) DeleteFirewallRule(gctx echo.Context) error {
	serverID, _ := strconv.ParseInt(gctx.FormValue("server_id"), 10, 64)
	ruleNumber, _ := strconv.ParseInt(gctx.FormValue("rule_number"), 10, 64)
	sessUser := c.GetSessionUser(gctx)
	rule := gctx.FormValue("rule")

	if !models.IsMyServer(serverID, sessUser.TeamId) {
		return gctx.Redirect(http.StatusFound, "/denied")
	}

	response := make(map[string]string)

	if serverID == 0 || ruleNumber == 0 {
		response["status"] = "failed"
		response["msg"] = "Bad server or rule ID."
		return gctx.JSON(http.StatusBadRequest, response)
	}

	server := models.GetServer(serverID, sessUser.TeamId)
	if server.ID == 0 {
		response["status"] = "failed"
		response["msg"] = "Bad server or rule ID."
		return gctx.JSON(http.StatusBadRequest, response)
	}

	err := models.DeleteFirewallRule(&server, ruleNumber, rule)

	if err != nil {
		response["status"] = "failed"
		response["msg"] = "Bad server or rule ID."
		return gctx.JSON(http.StatusBadRequest, response)
	}

	response["status"] = "success"
	response["msg"] = "Successfully deleted rule."

	return gctx.JSON(http.StatusOK, response)
}

func (c *Controller) AddFirewallRule(gctx echo.Context) error {
	serverID, _ := strconv.ParseInt(gctx.FormValue("server_id"), 10, 64)
	sessUser := c.GetSessionUser(gctx)
	rule := gctx.FormValue("rule")

	if !models.IsMyServer(serverID, sessUser.TeamId) {
		return gctx.Redirect(http.StatusFound, "/denied")
	}

	response := make(map[string]string)

	if serverID == 0 || rule == "" {
		response["status"] = "failed"
		response["msg"] = "Bad server or firewall rule."
		return gctx.JSON(http.StatusBadRequest, response)
	}

	server := models.GetServer(serverID, sessUser.TeamId)
	if server.ID == 0 {
		response["status"] = "failed"
		response["msg"] = "Bad server ID."
		return gctx.JSON(http.StatusBadRequest, response)
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

	err := models.AddFirewallRule(&server, rule)

	if err != nil {
		response["status"] = "failed"
		response["msg"] = "Failed to add firewall rule. Please try again."
		return gctx.JSON(http.StatusBadRequest, response)
	}

	response["status"] = "success"
	response["msg"] = "Successfully added firewall rule."
	return gctx.JSON(http.StatusOK, response)
}
