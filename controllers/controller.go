// This package handles all the controller logic for Scriptables, controller.go is the base controller and does the following:
//
// 1) Template rendering - we use Jinja templates: https://github.com/noirbizarre/gonja
//
// 2) Flash messages - you can show messages between routes, errors or success messages.
//
// 3) Session - allows for getting the logged in users session ID, and managing CSRF protection.
//
// 4) DB - access the DB from any controller: db := c.GetDB(gctx)
//
// The rest of the package is broken up into files for CRUD operations, and are named appropriately.
//
// Please note: logs.go manages access to log files. All logs are encrypted before being stored in the
// backend DB hence this is the only set of URLs that will decrypt and present logs in plain text.
package controllers

import (
	"fmt"
	"net/http"
	"os"

	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/noirbizarre/gonja"
	"plexcorp.tech/scriptable/models"
	"plexcorp.tech/scriptable/utils"
)

type Controller struct {
}

func (c *Controller) RenderHtml(tpl_name string, ctx gonja.Context, gctx echo.Context, layoutTpl string) error {

	sess, err := session.Get("session", gctx)
	flash, exists := sess.Values["success"]
	sessionChanged := false
	if exists {
		ctx["successMsg"] = flash.(string)
		sess.Values["success"] = ""
		sessionChanged = true
	}

	flash, exists = sess.Values["errors"]
	if exists {
		errors, _ := flash.(string)
		ctx["errors"] = []string{errors}
		sess.Values["errors"] = ""
		sessionChanged = true
	}

	if sessionChanged {
		sess.Save(gctx.Request(), gctx.Response())
	}

	_, ok := ctx["highlight"]
	if !ok {
		ctx["highlight"] = ""
	}

	ctx["scriptable_base_url"] = os.Getenv("SCRIPTABLE_URL")
	ctx["STATUS_QUEUED"] = models.STATUS_QUEUED
	ctx["STATUS_RUNNING"] = models.STATUS_RUNNING
	ctx["STATUS_FAILED"] = models.STATUS_FAILED
	ctx["STATUS_CONNECTING"] = models.STATUS_CONNECTING
	ctx["STATUS_COMPLETE"] = models.STATUS_COMPLETE

	view, err := gonja.Must(gonja.FromFile("templates/" + tpl_name + ".jinja")).Execute(ctx)
	if err != nil && utils.LogVerbose() {
		fmt.Println(err)
	}

	ctx["view"] = view

	var MASTER_TPL = gonja.Must(gonja.FromFile(layoutTpl + ".jinja"))
	tpl, err := MASTER_TPL.Execute(ctx)
	if err != nil && utils.LogVerbose() {
		fmt.Println(err)
	}

	return gctx.HTML(http.StatusOK, tpl)
}

// Renders templates to authenticated users only.
func (c *Controller) Render(tpl_name string, ctx gonja.Context, gctx echo.Context) error {
	return c.RenderHtml(tpl_name, ctx, gctx, "templates/master")
}

// This Render method renders public templates where authentication is not required.
func (c *Controller) RenderAuth(tpl_name string, ctx gonja.Context, gctx echo.Context) {
	c.RenderHtml(tpl_name, ctx, gctx, "templates/auth")
}

// Render plain text, mostly used for viewing logs.
func (c *Controller) RenderWithoutLayout(tpl_name string, ctx gonja.Context, gctx echo.Context) error {
	ctx["scriptable_base_url"] = os.Getenv("SCRIPTABLE_URL")
	view, err := gonja.Must(gonja.FromFile("templates/" + tpl_name + ".jinja")).Execute(ctx)
	if err != nil && utils.LogVerbose() {
		fmt.Println(err)
	}
	return gctx.HTML(http.StatusOK, view)
}

func (c *Controller) FlashMessage(gctx echo.Context, msg string, msgType string) {
	sess, _ := session.Get("session", gctx)
	sess.Values[msgType] = msg
	sess.Save(gctx.Request(), gctx.Response())
}

func (c *Controller) FlashSuccess(gctx echo.Context, msg string) {
	c.FlashMessage(gctx, msg, "error")
}

func (c *Controller) FlashError(gctx echo.Context, msg string) {
	c.FlashMessage(gctx, msg, "error")
}

func (c *Controller) GetSessionUser(gctx echo.Context) models.User {
	sess, _ := session.Get("session", gctx)
	userId, ok := sess.Values["user_id"]
	var user models.User
	if !ok {
		return user
	}
	c.GetDB(gctx).Raw("SELECT id, name, email, verified, team_id FROM users where id = ?", userId.(int64)).Scan(&user)
	return user
}

func (c *Controller) ShowGuide(gctx echo.Context) error {
	return c.Render("general/guide", gonja.Context{
		"title":     "Scriptables help guide",
		"highlight": "help",
	}, gctx)
}

func (c *Controller) AccessDenied(gctx echo.Context) error {
	return c.Render("general/permission", gonja.Context{
		"highlight": "",
	}, gctx)
}

func (c *Controller) TrialExpired(gctx echo.Context) error {
	return c.Render("general/trial_expired", gonja.Context{
		"highlight": "",
	}, gctx)
}
