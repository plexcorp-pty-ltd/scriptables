package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/labstack/echo/v4"
	"github.com/noirbizarre/gonja"
	"plexcorp.tech/scriptable/models"
	"plexcorp.tech/scriptable/utils"
)

func (c *Controller) ServerLogs(gctx *gin.Context) {
	serverId, _ := strconv.ParseInt(gctx.Param("id"), 10, 64)
	page, err := strconv.Atoi(gctx.Query("page"))
	sessUser := c.GetSessionUser(gctx)

	if err != nil {
		page = 1
	}

	perPage, err := strconv.Atoi(gctx.Query("perPage"))
	if err != nil {
		perPage = 20
	}

	logLevel := gctx.Query("log_level")
	logLevelQuery := ""
	if logLevel != "" {
		logLevelQuery = "&log_level=" + logLevel
	}

	logs := models.GetOperationLogs(c.GetDB(gctx), page, perPage, "server", serverId, logLevel, sessUser.TeamId)

	var server models.Server
	c.GetDB(gctx).Where("id", serverId).Where("team_id=?", sessUser.TeamId).Find(&server)
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

	c.Render("logs/list", vars, gctx)

}

func (c *Controller) ServerLogView(gctx echo.Context) error {
	serverId, _ := strconv.ParseInt(gctx.Param("id"), 10, 64)
	var log models.OperationLog
	sessUser := c.GetSessionUser(gctx)

	c.GetDB(gctx).Where("id = ? and entity='server' and team_id=?", serverId, sessUser.TeamId).First(&log)
	log.Log = utils.Decrypt(log.Log)

	return c.RenderWithoutLayout("logs/view_log", gonja.Context{
		"log":       log.Log,
		"highlight": "servers",
	}, gctx)

}

func (c *Controller) SiteLogs(gctx echo.Context) error {
	siteId, _ := strconv.ParseInt(gctx.Param("id"), 10, 64)
	sessUser := c.GetSessionUser(gctx)
	page, err := strconv.Atoi(gctx.Query("page"))
	if err != nil {
		page = 1
	}

	perPage, err := strconv.Atoi(gctx.Query("perPage"))
	if err != nil {
		perPage = 20
	}

	logLevel := gctx.Query("log_level")
	logLevelQuery := ""
	if logLevel != "" {
		logLevelQuery = "&log_level=" + logLevel
	}

	logs := models.GetOperationLogs(c.GetDB(gctx), page, perPage, "site", siteId, logLevel, sessUser.TeamId)

	var site models.Site
	c.GetDB(gctx).Where("id=? and team_id=?", siteId, sessUser.TeamId).Find(&site)
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

	var log models.OperationLog
	c.GetDB(gctx).Where("id = ? and entity='site' and team_id=?", siteID, sessUser.TeamId).First(&log)
	log.Log = utils.Decrypt(log.Log)

	site := models.GetSiteById(c.GetDB(gctx), log.EntityID, sessUser.TeamId)
	c.RenderWithoutLayout("logs/view_log", gonja.Context{
		"log":       log.Log,
		"highlight": "sites",
		"site":      site}, gctx)

}

func (c *Controller) CronLogs(gctx *gin.Context) {
	cronId, _ := strconv.ParseInt(gctx.Param("id"), 10, 64)
	page, err := strconv.Atoi(gctx.Query("page"))
	sessUser := c.GetSessionUser(gctx)

	if err != nil {
		page = 1
	}

	perPage, err := strconv.Atoi(gctx.Query("perPage"))
	if err != nil {
		perPage = 20
	}

	logLevel := gctx.Query("log_level")
	logLevelQuery := ""
	if logLevel != "" {
		logLevelQuery = "&log_level=" + logLevel
	}

	logs := models.GetOperationLogs(c.GetDB(gctx), page, perPage, "cron", cronId, logLevel, sessUser.TeamId)

	var cron models.Cron
	c.GetDB(gctx).Where("id=? and team_id=?", cronId, sessUser.TeamId).Find(&cron)
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

	c.Render("logs/cron_list", vars, gctx)
}

func (c *Controller) CronLogView(gctx *gin.Context) {
	cronId, _ := strconv.ParseInt(gctx.Param("id"), 10, 64)
	sessUser := c.GetSessionUser(gctx)

	var log models.OperationLog
	c.GetDB(gctx).Where("id = ? and entity='cron' and team_id=?", cronId, sessUser.TeamId).First(&log)
	log.Log = utils.Decrypt(log.Log)

	if log.ID == 0 {
		c.FlashError(gctx, "Sorry, log not found.")
		gctx.Redirect(http.StatusFound, "/crons")
	}

	c.RenderWithoutLayout("logs/view_log", gonja.Context{
		"log":       log.Log,
		"highlight": "crons",
	}, gctx)

}
