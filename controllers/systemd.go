package controllers

import (
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/noirbizarre/gonja"
	"plexcorp.tech/scriptable/models"
)

func (c *Controller) ListServices(gctx echo.Context) error {
	siteId, _ := strconv.ParseInt(gctx.Param("id"), 10, 64)
	sessUser := c.GetSessionUser(gctx)

	site := models.GetSiteById(siteId, sessUser.TeamId)
	services := models.GetSiteWorkers(site.ID, sessUser.TeamId)

	return c.Render("systemd/list", gonja.Context{
		"title":     site.Domain + " Queue Workers",
		"highlight": "sites",
		"services":  services,
		"site":      site,
	}, gctx)
}
