package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/labstack/echo/v4"
	"github.com/noirbizarre/gonja"
	"plexcorp.tech/scriptable/models"
	"plexcorp.tech/scriptable/utils"
)

func (c *Controller) CreateCron(gctx echo.Context) error {
	var countServers int64
	sessUser := c.GetSessionUser(gctx)

	c.GetDB(gctx).Table("servers").Where(
		"status=? and team_id=?", models.STATUS_COMPLETE, sessUser.TeamId).Count(&countServers)
	if countServers == 0 {
		return c.Render("general/warning", gonja.Context{
			"title":      "No active servers found",
			"highlight":  "crons",
			"warningMsg": "Please setup a server <a href=\"/\"> here</a> first before trying to setup crons. If you have already done so - please wait for the server build to finish first.",
		}, gctx)
	}

	servers := []models.Server{}
	c.GetDB(gctx).Where("team_id=?", sessUser.TeamId).Find(&servers)

	return c.Render("crons/form", gonja.Context{
		"title":           "Setup cron",
		"cron_expression": "* * * * *",
		"task":            "",
		"servers":         servers,
		"user":            "root",
		"cron_name":       "",
		"status":          "pending",
		"server_id":       0,
		"action":          "/cron/save/",
		"highlight":       "crons",
	}, gctx)

}

func (c *Controller) SaveCron(gctx echo.Context) error {
	sessUser := c.GetSessionUser(gctx)

	user := gctx.FormValue("user")
	task := gctx.FormValue("task")
	cron_expression := gctx.FormValue("cron_expression")
	cron_name := gctx.FormValue("cron_name")
	server_id, _ := strconv.ParseInt(gctx.FormValue("server_id"), 10, 64)
	errors := []string{}
	servers := []models.Server{}
	c.GetDB(gctx).Where("team_id=?", sessUser.TeamId).Find(&servers)

	ctx := gonja.Context{

		"title":           "Setup cron",
		"user":            user,
		"task":            task,
		"cron_expression": cron_expression,
		"cron_name":       cron_name,
		"servers":         servers,
		"status":          models.STATUS_QUEUED,
		"server_id":       server_id,
		"action":          "/cron/save/",
		"highlight":       "crons",
	}

	if !models.IsValidCronExpression(cron_expression) {
		errors = append(errors, "Sorry, the cron expression entered is invalid.")
	}

	if server_id == 0 {
		errors = append(errors, "Sorry, please select which server to deploy this cron.")
	}

	if task == "" {
		errors = append(errors, "Command to execute cannot be empty.")
	}

	if user == "" {
		user = "root"
	}

	if len(errors) == 0 {
		sessUser := c.GetSessionUser(gctx)
		cron := models.Cron{
			ServerID:       server_id,
			User:           user,
			Task:           task,
			Status:         models.STATUS_QUEUED,
			CronExpression: cron_expression,
			CronName:       cron_name,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
			TeamID:         sessUser.TeamId,
		}
		err := c.GetDB(gctx).Create(&cron)

		if err != nil && utils.LogVerbose() {
			fmt.Println(err)
		}

		c.FlashSuccess(gctx, "Successfully queued cron for deployment. Please check the logs for progress.")
		gctx.Redirect(http.StatusFound, "/crons")
		return
	} else {
		ctx["errors"] = errors
	}

	c.Render("crons/form", ctx, gctx)

}

func (c *Controller) Crons(gctx *gin.Context) {
	page, err := strconv.Atoi(gctx.Query("page"))
	sessUser := c.GetSessionUser(gctx)

	if err != nil {
		page = 1
	}

	perPage, err := strconv.Atoi(gctx.Query("perPage"))
	if err != nil {
		perPage = 20
	}

	search := gctx.Query("search")
	crons := models.GetCrons(c.GetDB(gctx), page, perPage, search, sessUser.TeamId)
	searchQuery := ""

	if search != "" {
		searchQuery = "&search=" + searchQuery
	}

	vars := gonja.Context{
		"title":       "Cron Jobs",
		"crons":       crons,
		"nextPage":    page + 1,
		"prevPage":    page - 1,
		"searchQuery": searchQuery,
		"search":      search,
		"numCrons":    len(crons),
		"highlight":   "crons",
		"addBtn":      "<a href=\"/crons/create\" class=\"btn-sm btn-success\" style=\"vertical-align:middle;\">ADD Cron</a>",
	}

	c.Render("crons/list", vars, gctx)

}

func (c *Controller) EditCron(gctx *gin.Context) {
	cronId, _ := strconv.ParseInt(gctx.Param("id"), 10, 64)
	sessUser := c.GetSessionUser(gctx)
	var cron models.Cron

	if cronId != 0 {
		c.GetDB(gctx).Where("id=?", cronId).Where("team_id=?", sessUser.TeamId).First(&cron)
	}

	if cron.ID == 0 {
		c.FlashError(gctx, "Cron with ID "+gctx.Param("id")+" does not exist.")
		gctx.Redirect(http.StatusFound, "/crons")
		return
	}

	servers := []models.Server{}
	c.GetDB(gctx).Where("team_id=?", sessUser.TeamId).Find(&servers)

	c.Render("crons/form", gonja.Context{
		"title":           "Setup cron",
		"cron_expression": cron.CronExpression,
		"task":            cron.Task,
		"servers":         servers,
		"user":            cron.User,
		"cron_name":       cron.CronName,
		"status":          cron.Status,
		"server_id":       0,
		"action":          fmt.Sprintf("/cron/update/%d", cron.ID),
		"highlight":       "crons",
	}, gctx)
}

func (c *Controller) UpdateCron(gctx *gin.Context) {
	user := gctx.FormValue("user")
	task := gctx.FormValue("task")
	cron_expression := gctx.FormValue("cron_expression")
	cron_name := gctx.FormValue("cron_name")
	server_id, _ := strconv.ParseInt(gctx.FormValue("server_id"), 10, 64)
	errors := []string{}
	servers := []models.Server{}
	sessUser := c.GetSessionUser(gctx)

	c.GetDB(gctx).Where("team_id=?", sessUser.TeamId).Find(&servers)

	cronId, _ := strconv.ParseInt(gctx.Param("id"), 10, 64)
	var cron models.Cron

	if cronId != 0 {
		c.GetDB(gctx).Where("id=?", cronId).Where("team_id =?", sessUser.TeamId).First(&cron)
	}

	if cron.ID == 0 {
		c.FlashError(gctx, "Cron with ID "+gctx.Param("id")+" does not exist.")
		gctx.Redirect(http.StatusFound, "/crons")
		return
	}

	ctx := gonja.Context{
		"title":           "Setup cron",
		"user":            user,
		"task":            task,
		"cron_expression": cron_expression,
		"cron_name":       cron_name,
		"servers":         servers,
		"status":          models.STATUS_QUEUED,
		"server_id":       server_id,
		"action":          fmt.Sprintf("/cron/update/%d", cron.ID),
		"highlight":       "crons",
	}

	if !models.IsValidCronExpression(cron_expression) {
		errors = append(errors, "Sorry, the cron expression entered is invalid.")
	}

	if server_id == 0 {
		errors = append(errors, "Sorry, please select which server to deploy this cron.")
	}

	if task == "" {
		errors = append(errors, "Command to execute cannot be empty.")
	}

	if user == "" {
		user = "root"
	}

	if len(errors) == 0 {
		cron.ServerID = server_id
		cron.User = user
		cron.Task = task
		cron.Status = models.STATUS_QUEUED
		cron.UpdatedAt = time.Now()
		cron.CronName = cron_name
		cron.CronExpression = cron_expression

		err := c.GetDB(gctx).Save(&cron)

		if err != nil && utils.LogVerbose() {
			fmt.Println(err)
		}

		c.FlashSuccess(gctx, "Successfully updated cron.")
		gctx.Redirect(http.StatusFound, "/crons")
		return
	} else {
		ctx["errors"] = errors
	}

	c.Render("crons/form", ctx, gctx)
}

func (c *Controller) DisableCron(gctx *gin.Context) {
	cronId, _ := strconv.ParseInt(gctx.FormValue("id"), 10, 64)
	sessUser := c.GetSessionUser(gctx)

	if cronId == 0 {
		c.FlashError(gctx, "Invalid Cron ID - please try again.")
		gctx.Redirect(http.StatusFound, "/crons")
		return
	}

	c.GetDB(gctx).Exec("UPDATE crons SET deleted_at = NOW(), status = ? WHERE id = ? and team_id = ?",
		models.STATUS_QUEUED, cronId, sessUser.TeamId)
	c.FlashSuccess(gctx, "Successfully queued cron for deletion.")
	gctx.Redirect(http.StatusFound, "/crons")
}

func (c *Controller) RetryCronBuild(gctx *gin.Context) {
	retryBuild := gctx.FormValue("retryBuildId")
	sessUser := c.GetSessionUser(gctx)
	updated := false

	if retryBuild != "" {
		sid, err := strconv.ParseInt(retryBuild, 10, 64)
		if err == nil && sid != 0 {
			c.GetDB(gctx).Exec("UPDATE crons set status='queued' where id=? and team_id = ?", sid, sessUser.TeamId)
			updated = true
		}
	}

	if !updated {
		c.FlashError(gctx, "Sorry, failed to queue cron deploy. Please try again.")
	} else {
		c.FlashSuccess(gctx, "Successfully queued cron for deployment.")
	}

	gctx.Redirect(http.StatusFound, "/crons")
}
