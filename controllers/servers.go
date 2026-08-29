package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/noirbizarre/gonja"
	"plexcorp.tech/scriptable/models"
	"plexcorp.tech/scriptable/sshclient"
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
	retryBuild := gctx.PostForm("retryBuildServerId")
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
	if _, err := sshclient.LocalSigners(); err != nil {
		c.Render("general/warning", gonja.Context{
			"title":      "No usable SSH keys found",
			"warningMsg": "Scriptables uses the SSH keys already on this machine. " + err.Error(),
		}, gctx)

		return
	}

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

	signers, _ := sshclient.LocalSigners()

	vars := gonja.Context{
		"serverTypes":       models.GetServerTypes(),
		"sshDir":            sshclient.SshDir(),
		"sshKeyCount":       len(signers),
		"ServerName":        server.ServerName,
		"ServerType":        server.ServerType,
		"ServerIP":          server.ServerIP,
		"PrivateServerIP":   server.PrivateServerIP,
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
		"isExisting":        models.IsExistingServer(server.ServerType),
		"action":            "/server/create/" + server.ServerType,
		"actionType":        "BUILD SERVER",
		"title":             "Build new " + server.ServerType + " server",
		"highlight":         "servers",
	}

	if models.IsExistingServer(server.ServerType) {
		vars["actionType"] = "CONNECT SERVER"
		vars["title"] = "Connect an existing server"
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
	sessUser := c.GetSessionUser(gctx)
	db := c.GetDB(gctx)

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

	signers, _ := sshclient.LocalSigners()

	vars := gonja.Context{
		"serverTypes":       models.GetServerTypes(),
		"sshDir":            sshclient.SshDir(),
		"sshKeyCount":       len(signers),
		"ServerName":        server.ServerName,
		"ServerType":        server.ServerType,
		"ServerIP":          server.ServerIP,
		"PrivateServerIP":   server.PrivateServerIP,
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
		"isExisting":        models.IsExistingServer(server.ServerType),
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

// TestSSHConnection is called by htmx from the connection test page. On success
// it answers with an HX-Redirect so the browser moves straight to the build log;
// on failure it returns an inline alert fragment explaining what went wrong.
func (c *Controller) TestSSHConnection(gctx *gin.Context) {
	id, _ := strconv.ParseInt(gctx.Param("id"), 10, 64)
	sessUser := c.GetSessionUser(gctx)
	db := c.GetDB(gctx)

	if !models.IsMyServer(db, id, sessUser.TeamId) {
		gctx.Redirect(http.StatusFound, "/denied")
		return
	}

	server := models.GetServer(db, id, sessUser.TeamId)

	fail := func(reason string) {
		db.Exec("UPDATE servers set status=? WHERE id = ?", models.STATUS_FAILED, id)
		c.RenderWithoutLayout("servers/_sshtest_result", gonja.Context{
			"id":       id,
			"errorMsg": reason,
		}, gctx)
	}

	if server.ID == 0 {
		fail("That server could not be found.")
		return
	}

	connection, err := models.GetSSHClient(&server, true)
	if err != nil {
		fail("Could not open an SSH connection to " + server.ServerIP + ": " + err.Error())
		return
	}

	connection.Close()

	// An existing server is left exactly as it is. Marking it complete keeps the
	// build daemon, which only picks up queued servers, away from it.
	if models.IsExistingServer(server.ServerType) {
		db.Exec("UPDATE servers SET status=? WHERE id=?", models.STATUS_COMPLETE, server.ID)
		c.FlashSuccess(gctx, "Connected to "+server.ServerName+". The server was left untouched.")
		gctx.Header("HX-Redirect", "/servers")
		gctx.Status(http.StatusOK)
		return
	}

	db.Exec("UPDATE servers SET status=? WHERE id=?", models.STATUS_QUEUED, server.ID)

	gctx.Header("HX-Redirect", fmt.Sprintf("/logs/server/%d", server.ID))
	gctx.Status(http.StatusOK)
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

// renderFirewallRules returns the rules table fragment that htmx swaps into the
// page. Every firewall action ends here so the list always reflects the real
// ufw state rather than something patched together in the browser.
func (c *Controller) renderFirewallRules(gctx *gin.Context, serverID int64, successMsg string, actionErr error) {
	db := c.GetDB(gctx)
	sessUser := c.GetSessionUser(gctx)

	vars := gonja.Context{"serverID": serverID}
	if successMsg != "" {
		vars["successMsg"] = successMsg
	}

	rules := []models.FirewallRule{}
	var fetchErr error

	server := models.GetServer(db, serverID, sessUser.TeamId)
	if server.ID == 0 {
		fetchErr = errors.New("Invalid server ID.")
	} else if client, err := models.GetSSHClient(&server, false); err != nil {
		fetchErr = err
	} else if fetched, err := models.GetRules(client); err != nil {
		fetchErr = err
	} else {
		rules = fetched
	}

	// The action's own failure is what the user needs to see; a follow up fetch
	// error would otherwise mask it.
	if actionErr != nil {
		vars["errorMsg"] = actionErr.Error()
	} else if fetchErr != nil {
		vars["errorMsg"] = fetchErr.Error()
	}

	vars["rules"] = rules
	vars["numRules"] = len(rules)

	c.RenderWithoutLayout("servers/_firewall_rules", vars, gctx)
}

func (c *Controller) FirewallRulesAjax(gctx *gin.Context) {
	id, err := strconv.ParseInt(gctx.Param("serverID"), 10, 64)
	db := c.GetDB(gctx)
	sessUser := c.GetSessionUser(gctx)

	if err != nil || !models.IsMyServer(db, id, sessUser.TeamId) {
		gctx.Redirect(http.StatusFound, "/denied")
		return
	}

	c.renderFirewallRules(gctx, id, "", nil)
}

func (c *Controller) DeleteFirewallRule(gctx *gin.Context) {
	serverID, _ := strconv.ParseInt(gctx.PostForm("server_id"), 10, 64)
	ruleNumber, _ := strconv.ParseInt(gctx.PostForm("rule_number"), 10, 64)
	sessUser := c.GetSessionUser(gctx)
	rule := gctx.PostForm("rule")
	db := c.GetDB(gctx)

	if !models.IsMyServer(db, serverID, sessUser.TeamId) {
		gctx.Redirect(http.StatusFound, "/denied")
		return
	}

	if serverID == 0 || ruleNumber == 0 {
		c.renderFirewallRules(gctx, serverID, "", errors.New("Bad server or rule number."))
		return
	}

	server := models.GetServer(db, serverID, sessUser.TeamId)
	if server.ID == 0 {
		c.renderFirewallRules(gctx, serverID, "", errors.New("Bad server or rule number."))
		return
	}

	if err := models.DeleteFirewallRule(db, &server, ruleNumber, rule); err != nil {
		c.renderFirewallRules(gctx, serverID, "", err)
		return
	}

	c.renderFirewallRules(gctx, serverID, "Successfully deleted the rule.", nil)
}

// buildUfwRule composes a ufw rule from the form fields. The browser used to
// assemble this string itself; doing it here keeps the shell command in one
// place and lets the fields be validated.
func buildUfwRule(allowBlock, direction, ip, port, protocol string) string {
	var rule string
	if direction == "outgoing" {
		rule = fmt.Sprintf("%s out to %s port %s proto %s", allowBlock, ip, port, protocol)
	} else {
		rule = fmt.Sprintf("%s from %s to any port %s proto %s", allowBlock, ip, port, protocol)
	}

	rule = strings.ToLower(rule)

	if strings.Contains(rule, "anywhere") {
		rule = strings.ReplaceAll(rule, "to anywhere port", "")
		rule = strings.ReplaceAll(rule, "from anywhere to any port", "")
		rule = strings.ReplaceAll(rule, "  ", " ")
		rule = strings.ReplaceAll(rule, " proto tcp", "/tcp")
		rule = strings.ReplaceAll(rule, " proto udp", "/udp")
	}

	return strings.TrimSpace(strings.ReplaceAll(rule, "port any proto", "proto"))
}

func (c *Controller) AddFirewallRule(gctx *gin.Context) {
	serverID, _ := strconv.ParseInt(gctx.PostForm("server_id"), 10, 64)
	sessUser := c.GetSessionUser(gctx)
	db := c.GetDB(gctx)

	if !models.IsMyServer(db, serverID, sessUser.TeamId) {
		gctx.Redirect(http.StatusFound, "/denied")
		return
	}

	allowBlock := gctx.PostForm("allow_block")
	direction := gctx.PostForm("direction")
	ip := strings.TrimSpace(gctx.PostForm("ip"))
	port := strings.TrimSpace(gctx.PostForm("port"))
	protocol := gctx.PostForm("protocol")

	if serverID == 0 || allowBlock == "" || direction == "" || ip == "" || port == "" || protocol == "" {
		c.renderFirewallRules(gctx, serverID, "", errors.New("Please fill in every field before adding a rule."))
		return
	}

	server := models.GetServer(db, serverID, sessUser.TeamId)
	if server.ID == 0 {
		c.renderFirewallRules(gctx, serverID, "", errors.New("Bad server ID."))
		return
	}

	rule := buildUfwRule(allowBlock, direction, ip, port, protocol)

	if err := models.AddFirewallRule(db, &server, rule); err != nil {
		c.renderFirewallRules(gctx, serverID, "", fmt.Errorf("Failed to add rule %q: %s", rule, err))
		return
	}

	c.renderFirewallRules(gctx, serverID, "Successfully added the rule.", nil)
}
