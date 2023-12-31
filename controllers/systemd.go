package controllers

import (
	"strconv"

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
		"title":     site.Domain + " Queue Workers",
		"highlight": "sites",
		"services":  services,
		"site":      site,
	}, gctx)
}
