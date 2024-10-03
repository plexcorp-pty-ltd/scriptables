package controllers

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/noirbizarre/gonja"
	"plexcorp.tech/scriptable/console"
	"plexcorp.tech/scriptable/models"
	"plexcorp.tech/scriptable/sshclient"
	"plexcorp.tech/scriptable/utils"
)

func (c *Controller) CreateSite(gctx echo.Context) error {
	var countServers int64
	sessUser := c.GetSessionUser(gctx)
	db := models.GetDB()

	db.Table("servers").Where("status=? AND team_id=?", models.STATUS_COMPLETE, sessUser.TeamId).Count(&countServers)
	if countServers == 0 {
		return c.Render("general/warning", gonja.Context{
			"title":      "No active servers found",
			"warningMsg": "Please setup a server <a href=\"/\"> here</a> first before trying to deploy a site. If you have already done so - please wait for the server build to finish first.",
		}, gctx)
	}
	servers := []models.Server{}
	db.Where("team_id=?", sessUser.TeamId).Find(&servers)

	sshKeys := []models.SshKey{}
	db.Where("team_id=?", sessUser.TeamId).Find(&sshKeys)

	password := utils.GenPassword()

	return c.Render("sites/create", gonja.Context{
		"title":                   "Setup website",
		"server_id":               0,
		"domain":                  "",
		"webroot":                 "public",
		"php_version":             "",
		"letsencrypt_certificate": 0,
		"git_url":                 "",
		"scriptables":             "laravel",
		"deploy_scriptables":      "laraveldeploy",
		"mysql_password":          password,
		"mysql_password_confirm":  password,
		"sshKeys":                 sshKeys,
		"servers":                 servers,
		"environment":             "prod",
		"branch":                  "master",
		"highlight":               "sites",
	}, gctx)

}

func (c *Controller) SaveSite(gctx echo.Context) error {
	domain := gctx.FormValue("domain")
	serverId, serr := strconv.ParseInt(gctx.FormValue("server_id"), 10, 64)
	webroot := gctx.FormValue("webroot")
	giturl := gctx.FormValue("git_url")
	PhpVersion := gctx.FormValue("php_version")
	scriptables := gctx.FormValue("scriptables")
	MysqlPassword := gctx.FormValue("mysql_password")
	MysqlPasswordConfirm := gctx.FormValue("mysql_password_confirm")
	environment := gctx.FormValue("environment")
	branch := gctx.FormValue("branch")

	db := models.GetDB()

	LetsEncryptCertificate := 0
	servers := []models.Server{}
	db.Find(&servers)

	if gctx.FormValue("letsencrypt_certificate") != "" && gctx.FormValue("letsencrypt_certificate") == "on" {
		LetsEncryptCertificate = 1
	}

	var unwantedUrlsParts = []string{"https://", "http://", "://", "/"}
	for _, un := range unwantedUrlsParts {
		domain = strings.ReplaceAll(domain, un, "")
	}

	var siteName = ""
	if domain != "" {
		siteName = strings.ReplaceAll(domain, ".", "")
	}

	ctx := gonja.Context{
		"title":                   "Setup a website",
		"domain":                  domain,
		"server_id":               serverId,
		"webroot":                 webroot,
		"php_version":             PhpVersion,
		"letsencrypt_certificate": LetsEncryptCertificate,
		"git_url":                 giturl,
		"servers":                 servers,
		"site_name":               siteName,
		"mysql_password":          MysqlPassword,
		"mysql_password_confirm":  MysqlPasswordConfirm,
		"highlight":               "sites",
	}

	errors := []string{}

	if scriptables == "" {
		scriptables = "laravel"
	}

	deploy_scriptables := scriptables + "_deploy"

	if environment == "" {
		environment = "prod"
	}

	if branch == "" {
		branch = "master"
	}

	ctx["branch"] = branch
	ctx["scriptables"] = scriptables
	ctx["deploy_scriptables"] = deploy_scriptables
	ctx["environment"] = environment

	if siteName == "" {
		errors = append(errors, "Domain seems invalid. Please check the domain uses this format: domain.com|.ext or www.domain.ext or subdomain.domain.ext")
	}

	if MysqlPassword != "" && MysqlPassword != MysqlPasswordConfirm {
		errors = append(errors, "Mysql password and confirm password not the same.")
	}

	if giturl == "" {
		errors = append(errors, "Please enter a valid GIT URL.")
	}

	if strings.Contains(giturl, "https://") {
		errors = append(errors, "Please use only the SSH GIT URL. e.g.: git@github.com:username/app.git")
	}

	if serverId == 0 || serr != nil {
		errors = append(errors, "Please select a server to deploy this application to.")
	}

	if domain == "" {
		errors = append(errors, "Please enter a valid domain name.")
	}

	if webroot == "" {
		errors = append(errors, "Please enter the full path to your websites web root folder.")
	}

	if PhpVersion == "" {
		errors = append(errors, "Please select a version of PHP to configure with this app.")
	}

	var found int64
	db.Where("domain=?", domain, siteName).Count(&found)

	if found > 0 {
		errors = append(errors, "Sorry, domain already in use. You can have multiple subdomains but only one root domain.")
	}

	if len(errors) == 0 {
		token := uuid.New()
		sessUser := c.GetSessionUser(gctx)
		site := models.Site{
			Domain:                 domain,
			SiteName:               siteName,
			ServerID:               serverId,
			PhpVersion:             PhpVersion,
			Webroot:                webroot,
			LetsEncryptCertificate: LetsEncryptCertificate,
			Status:                 models.STATUS_CONNECTING,
			ScriptableName:         scriptables,
			DeployScriptables:      deploy_scriptables,
			GitURL:                 giturl,
			MysqlPassword:          utils.Encrypt(MysqlPassword),
			CreatedAt:              time.Now(),
			UpdatedAt:              time.Now(),
			Environment:            environment,
			Branch:                 branch,
			DeployToken:            strings.ReplaceAll(token.String(), "-", ""),
			TeamId:                 sessUser.TeamId,
		}
		err := db.Create(&site)

		if err != nil && utils.LogVerbose() {
			fmt.Println(err)
		}

		return gctx.Redirect(http.StatusFound, "/site/deployKey/"+strconv.Itoa(int(site.ID)))

	} else {
		ctx["errors"] = errors
	}

	return c.Render("sites/create", ctx, gctx)

}

func (c *Controller) Sites(gctx echo.Context) error {

	view := gctx.QueryParam("view")
	status := gctx.QueryParam("status")
	sessUser := c.GetSessionUser(gctx)
	if view == "" {
		view = "grid"
	}

	if status == "" {
		status = "all"
	}

	page, err := strconv.Atoi(gctx.QueryParam("page"))
	if err != nil {
		page = 1
	}

	perPage, err := strconv.Atoi(gctx.QueryParam("perPage"))
	if err != nil {
		perPage = 20
	}

	search := gctx.QueryParam("search")
	sites := models.GetSitesList(page, perPage, search, status, sessUser.TeamId)
	searchQuery := ""

	if search != "" {
		searchQuery = "&search=" + searchQuery
	}

	vars := gonja.Context{
		"title":       "Sites",
		"sites":       sites,
		"nextPage":    page + 1,
		"prevPage":    page - 1,
		"searchQuery": searchQuery,
		"search":      search,
		"view":        view,
		"status":      status,
		"numSites":    len(sites),
		"addBtn":      "<a href=\"/site/create\" class=\"btn-sm btn-success\" style=\"vertical-align:middle;\">ADD Site</a>",
		"highlight":   "sites",
	}

	return c.Render("sites/list", vars, gctx)
}

func (c *Controller) CreateSiteDeployKey(gctx echo.Context) error {
	siteId := gctx.Param("id")
	var site models.Site
	sessUser := c.GetSessionUser(gctx)

	models.GetDB().Where("id=? and team_id=?", siteId, sessUser.TeamId).First(&site)
	if site.ID == 0 || site.TeamId != sessUser.TeamId {
		c.FlashError(gctx, "Ooops, sorry seems like you do not have permission to access this site. Please try again.")
		return gctx.Redirect(http.StatusFound, "/sites")
	}

	return c.Render("sites/deploykey", gonja.Context{
		"title":      "GIT setup for: " + site.SiteName,
		"siteId":     site.ID,
		"token":      site.DeployToken,
		"highlight":  "sites",
		"successMsg": "Successfully saved site: " + site.SiteName + ". Now generating deploy key..., once done please copy and add to your repos deploy keys.",
	}, gctx)
}

func (c *Controller) GenerateDeployKey(gctx echo.Context) error {
	type Response struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
		PubKey  string `json:"pubkey"`
	}

	siteId, e := strconv.ParseInt(gctx.FormValue("siteId"), 10, 64)

	if siteId == 0 || e != nil {
		msg := Response{Success: false, Error: "Please specify a site key"}
		return gctx.JSON(400, msg)
	}

	var site models.Site
	sessUser := c.GetSessionUser(gctx)

	db := models.GetDB()
	db.Where("id=? and team_id = ?", siteId, sessUser.TeamId).First(&site)

	if site.ID == 0 || site.TeamId != sessUser.TeamId {
		msg := Response{Success: false, Error: "Sorry, seems like theres a permission issue. Please try again."}
		return gctx.JSON(400, msg)
	}

	server := models.GetServer(site.ServerID, site.TeamId)

	username := utils.Slugify(site.SiteName)
	keyPath := models.GetSiteDeployPubKeyPath(siteId, site.SiteName, username)
	cmd, err := utils.GetSharedScriptable("deploy_keysetup")

	if err != nil {
		msg := Response{Success: false, Error: "Failed to find deploy key setup script. Please check that a script named: deploy_keysetup.sh exists in scriptables/__shared/."}
		return gctx.JSON(400, msg)
	}

	cmd = site.SubScriptableSiteVarsOnly(cmd)

	client, err := models.GetSSHClient(&server, false)

	if client == nil || err != nil {
		msg := Response{Success: false, Error: "Failed to connect to server via SSH, please try again."}
		return gctx.JSON(400, msg)
	}

	summary := "Create deploy key:" + site.SiteName + " for server: " + server.ServerName
	err, output := models.RunScriptable("site", site.ID, client, cmd, summary, false, sessUser.TeamId)

	if utils.LogVerbose() {
		fmt.Println(err, output)
	}

	if err != nil {
		msg := Response{Success: false, Error: "Failed to create SSH key ` + keyPath + ` on server: ` + server.ServerName + `."}
		return gctx.JSON(400, msg)
	}

	pubKey, err := sshclient.ReadFileWithSudo(client, keyPath+".pub")
	publicKey := string(pubKey)
	if err != nil || publicKey == "" {
		if err != nil {
			msg := Response{Success: false, Error: "Failed to connect to create SSH key ` + keyPath + ` on server: ` + server.ServerName + `."}
			return gctx.JSON(400, msg)
		}
	}

	if err == nil {
		publicKey := base64.StdEncoding.EncodeToString([]byte(publicKey))

		msg := Response{Success: true, PubKey: publicKey}
		return gctx.JSON(200, msg)
	}

	msg := Response{Success: false, Error: "Failed to create SSH key ` + keyPath + ` on server: ` + server.ServerName + `."}
	return gctx.JSON(400, msg)

}

func (c *Controller) DeployBranch(gctx echo.Context) error {

	siteId, e := strconv.ParseInt(gctx.FormValue("siteId"), 10, 64)
	sessUser := c.GetSessionUser(gctx)

	if siteId == 0 || e != nil {
		c.FlashError(gctx, "Site ID is required")
		return gctx.Redirect(http.StatusFound, "/sites")
	}

	var site *models.Site
	db := models.GetDB()

	db.Where("id=? and team_id=?", siteId, sessUser.TeamId).First(&site)
	server := models.GetServer(site.ServerID, sessUser.TeamId)

	scripts := utils.GetScriptables(site.DeployScriptables)
	go console.RunSiteBuild(db, site, &server, scripts, false, nil)

	c.FlashSuccess(gctx, "Success! deploy will begin shortly...")
	return gctx.Redirect(http.StatusFound, fmt.Sprintf("/logs/site/%d", site.ID))
}

func (c *Controller) ConfirmSiteDeploy(gctx echo.Context) error {

	siteId, e := strconv.ParseInt(gctx.FormValue("siteId"), 10, 64)
	sessUser := c.GetSessionUser(gctx)

	if siteId == 0 || e != nil {
		c.FlashError(gctx, "Site ID is required")
		return gctx.Redirect(http.StatusFound, "/sites")
	}

	db := models.GetDB()

	db.Exec("UPDATE sites SET status = ? WHERE id = ? and team_id=?", models.STATUS_QUEUED, siteId, sessUser.TeamId)
	c.FlashSuccess(gctx, "Success! deploy will begin shortly...")
	return gctx.Redirect(http.StatusFound, fmt.Sprintf("/logs/site/%d", siteId))
}

func (c *Controller) RetrySiteBuild(gctx echo.Context) error {
	siteId, e := strconv.ParseInt(gctx.FormValue("siteId"), 10, 64)
	sessUser := c.GetSessionUser(gctx)

	if siteId == 0 || e != nil {
		c.FlashError(gctx, "Site ID is required")
		return gctx.Redirect(http.StatusFound, "/sites")
	}

	r := models.GetDB().Exec("Update sites set status=? WHERE id = ? and team_id=?",
		models.STATUS_QUEUED, siteId, sessUser.TeamId)
	if r.RowsAffected > 0 {
		c.FlashSuccess(gctx, "Successfully queued site for re-deploy. Please check the logs for more information.")
		return gctx.Redirect(http.StatusFound, fmt.Sprintf("/logs/site/%d", siteId))
	}

	c.FlashError(gctx, "Site ID is is invalid or an unknown error as occurred. Pleasy try again.")
	return gctx.Redirect(http.StatusFound, "/sites")
}
