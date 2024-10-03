package controllers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"plexcorp.tech/scriptable/models"
)

// Will trigger a deploy - useful to use with your version control system, to build
// and deploy a webapp on PUSH events.
func (c *Controller) DeployWebhookSite(gctx echo.Context) error {
	siteId, err := strconv.ParseInt(strings.Trim(gctx.Param("sid"), " "), 10, 64)
	token := strings.Trim(gctx.Param("token"), " ")

	if err != nil || token == "" || len(token) < 5 {
		return gctx.JSON(http.StatusBadRequest, "Invalid SID supplied")
	}

	db := models.GetDB()

	site := models.GetSiteByTokenAndId(token, siteId)

	if site.ID == 0 {
		return gctx.JSON(http.StatusBadRequest, "Invalid SID or token. Cannot find site.")
	}

	var siteQueue models.SiteQueue
	siteQueue.SiteID = site.ID
	siteQueue.Status = models.STATUS_QUEUED
	siteQueue.CreatedAt = time.Now()
	siteQueue.UpdatedAt = time.Now()

	db.Create(&siteQueue)

	response := make(map[string]string)

	if siteQueue.ID == 0 {
		response["status"] = "error"
		response["message"] = "Sorry deploy failed - please try again."
		return gctx.JSON(500, response)
	}

	response["status"] = "success"
	response["message"] = "Successfully queued deploy."
	return gctx.JSON(http.StatusOK, response)

}
