package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/noirbizarre/gonja"
	"plexcorp.tech/scriptable/models"
)

func (c *Controller) ListServices(gctx *gin.Context) {
	siteId, _ := strconv.ParseInt(gctx.Param("id"), 10, 64)
	sessUser := c.GetSessionUser(gctx)
	db := c.GetDB(gctx)

	site := models.GetSiteById(db, siteId, sessUser.TeamId)
	services := models.GetSiteWorkers(db, site.ID, sessUser.TeamId)

	c.Render("systemd/list", gonja.Context{
		"title":     site.Domain + " Systemd Daemons",
		"highlight": "sites",
		"services":  services,
		"site":      site,
	}, gctx)
}

func (c *Controller) CreateService(gctx *gin.Context) {
	siteId, _ := strconv.ParseInt(gctx.Param("id"), 10, 64)
	sessUser := c.GetSessionUser(gctx)
	db := c.GetDB(gctx)

	site := models.GetSiteById(db, siteId, sessUser.TeamId)

	c.Render("systemd/form", gonja.Context{
		"title":       "Add Systemd Daemon for: " + site.Domain,
		"highlight":   "sites",
		"site":        site,
		"command":     "artisan queue:work",
		"scriptables": "laravel_queues",
		"name":        "",
	}, gctx)
}

func (c *Controller) SaveService(gctx *gin.Context) {
	siteId, _ := strconv.ParseInt(gctx.Param("id"), 10, 64)
	sessUser := c.GetSessionUser(gctx)
	db := c.GetDB(gctx)

	site := models.GetSiteById(db, siteId, sessUser.TeamId)

	command := gctx.PostForm("command")
	scriptables := gctx.PostForm("scriptables")
	name := gctx.PostForm("name")

	hasErrors := false
	if command == "" || name == "" || scriptables == "" {
		hasErrors = true
		c.FlashError(gctx, "Sorry, all required fields with an * needs to be filled in.")
	}

	if !hasErrors {
		service := models.SystemdService{}
		service.Name = name
		service.Status = models.STATUS_QUEUED
		service.TeamId = sessUser.TeamId
		service.SiteID = site.ID
		service.Command = command
		service.Scriptables = scriptables
		service.CreatedAt = time.Now()
		service.UpdatedAt = time.Now()
		db.Create(&service)

		if service.ID != 0 {
			c.FlashSuccess(gctx, "Successfully queued to deploy. Please check the build log for more information.")
			gctx.Redirect(http.StatusFound, fmt.Sprintf("/logs/systemd/%d", service.ID))
			return
		} else {
			c.FlashError(gctx, "Sorry, something went wrong. Please try again.")
		}
	}

	c.Render("systemd/form", gonja.Context{
		"title":       "Add Systemd Daemon for: " + site.Domain,
		"highlight":   "sites",
		"site":        site,
		"command":     command,
		"scriptables": scriptables,
		"name":        name,
	}, gctx)
}
