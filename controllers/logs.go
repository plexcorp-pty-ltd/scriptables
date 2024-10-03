package controllers

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/noirbizarre/gonja"
	"plexcorp.tech/scriptable/models"
	"plexcorp.tech/scriptable/utils"
)

func (c *Controller) ServerLogs(gctx echo.Context) error {
	serverId, _ := strconv.ParseInt(gctx.Param("id"), 10, 64)
	page, err := strconv.Atoi(gctx.QueryParam("page"))
	sessUser := c.GetSessionUser(gctx)
	db := models.GetDB()

	if err != nil {
		page = 1
	}

	perPage, err := strconv.Atoi(gctx.QueryParam("perPage"))
	if err != nil {
		perPage = 20
	}

	logLevel := gctx.QueryParam("log_level")
	logLevelQuery := ""
	if logLevel != "" {
		logLevelQuery = "&log_level=" + logLevel
	}

	logs := models.GetOperationLogs(page, perPage, "server", serverId, logLevel, sessUser.TeamId)

	var server models.Server
	db.Where("id", serverId).Where("team_id=?", sessUser.TeamId).Find(&server)
	vars := gonja.Context{
		"title":         "Server logs for: " + server.ServerName,
		"logs":          logs,
		"log_level":     logLevel,
		"nextPage":      page + 1,
		"prevPage":      page - 1,
		"server":        server,
		"logLevelQuery": logLevelQuery,
		"highlight":     "servers",
	}

	return c.Render("logs/list", vars, gctx)

}

func (c *Controller) ServerLogView(gctx echo.Context) error {
	serverId, _ := strconv.ParseInt(gctx.Param("id"), 10, 64)
	var log models.OperationLog
	sessUser := c.GetSessionUser(gctx)
	db := models.GetDB()

	db.Where("id = ? and entity='server' and team_id=?", serverId, sessUser.TeamId).First(&log)
	log.Log = utils.Decrypt(log.Log)

	return c.RenderWithoutLayout("logs/view_log", gonja.Context{
		"log":       log.Log,
		"highlight": "servers",
	}, gctx)

}

func (c *Controller) SiteLogs(gctx echo.Context) error {
	siteId, _ := strconv.ParseInt(gctx.Param("id"), 10, 64)
	sessUser := c.GetSessionUser(gctx)
	page, err := strconv.Atoi(gctx.QueryParam("page"))
	if err != nil {
		page = 1
	}

	perPage, err := strconv.Atoi(gctx.QueryParam("perPage"))
	if err != nil {
		perPage = 20
	}

	db := models.GetDB()

	logLevel := gctx.QueryParam("log_level")
	logLevelQuery := ""
	if logLevel != "" {
		logLevelQuery = "&log_level=" + logLevel
	}

	logs := models.GetOperationLogs(page, perPage, "site", siteId, logLevel, sessUser.TeamId)

	var site models.Site
	db.Where("id=? and team_id=?", siteId, sessUser.TeamId).Find(&site)
	vars := gonja.Context{
		"title":         "Site logs for: " + site.SiteName,
		"logs":          logs,
		"log_level":     logLevel,
		"nextPage":      page + 1,
		"prevPage":      page - 1,
		"site":          site,
		"logLevelQuery": logLevelQuery,
		"highlight":     "sites",
	}

	return c.Render("logs/site_list", vars, gctx)

}

func (c *Controller) SiteLogView(gctx echo.Context) error {
	siteID, _ := strconv.ParseInt(gctx.Param("id"), 10, 64)
	sessUser := c.GetSessionUser(gctx)
	db := models.GetDB()

	var log models.OperationLog
	db.Where("id = ? and entity='site' and team_id=?", siteID, sessUser.TeamId).First(&log)
	log.Log = utils.Decrypt(log.Log)

	site := models.GetSiteById(log.EntityID, sessUser.TeamId)
	return c.RenderWithoutLayout("logs/view_log", gonja.Context{
		"log":       log.Log,
		"highlight": "sites",
		"site":      site}, gctx)
}

func (c *Controller) CronLogs(gctx echo.Context) error {
	cronId, _ := strconv.ParseInt(gctx.Param("id"), 10, 64)
	page, err := strconv.Atoi(gctx.QueryParam("page"))
	sessUser := c.GetSessionUser(gctx)
	db := models.GetDB()

	if err != nil {
		page = 1
	}

	perPage, err := strconv.Atoi(gctx.QueryParam("perPage"))
	if err != nil {
		perPage = 20
	}

	logLevel := gctx.QueryParam("log_level")
	logLevelQuery := ""
	if logLevel != "" {
		logLevelQuery = "&log_level=" + logLevel
	}

	logs := models.GetOperationLogs(page, perPage, "cron", cronId, logLevel, sessUser.TeamId)

	var cron models.Cron
	db.Where("id=? and team_id=?", cronId, sessUser.TeamId).Find(&cron)
	vars := gonja.Context{
		"title":         "Cron logs for: " + cron.CronName,
		"logs":          logs,
		"log_level":     logLevel,
		"nextPage":      page + 1,
		"prevPage":      page - 1,
		"cron":          cron,
		"logLevelQuery": logLevelQuery,
		"highlight":     "crons",
	}

	return c.Render("logs/cron_list", vars, gctx)
}

func (c *Controller) CronLogView(gctx echo.Context) error {
	cronId, _ := strconv.ParseInt(gctx.Param("id"), 10, 64)
	sessUser := c.GetSessionUser(gctx)
	db := models.GetDB()

	var log models.OperationLog
	db.Where("id = ? and entity='cron' and team_id=?", cronId, sessUser.TeamId).First(&log)
	log.Log = utils.Decrypt(log.Log)

	if log.ID == 0 {
		c.FlashError(gctx, "Sorry, log not found.")
		return gctx.Redirect(http.StatusFound, "/crons")
	}

	return c.RenderWithoutLayout("logs/view_log", gonja.Context{
		"log":       log.Log,
		"highlight": "crons",
	}, gctx)

}
