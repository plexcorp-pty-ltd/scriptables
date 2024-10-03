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

	flashes, err := c.GetFlashMessages(models.FLASH_SUCCESS, gctx)
	if err == nil && len(flashes) > 0 {
		ctx["success"] = flashes
	}

	flashes, err = c.GetFlashMessages(models.FLASH_SUCCESS, gctx)
	if err == nil && len(flashes) > 0 {
		ctx["errors"] = flashes
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

	view, err := gonja.Must(gonja.FromFile("templates/" + tpl_name + ".html")).Execute(ctx)
	if err != nil && utils.LogVerbose() {
		fmt.Println(err)
	}

	ctx["view"] = view

	var MASTER_TPL = gonja.Must(gonja.FromFile(layoutTpl + ".html"))
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
func (c *Controller) RenderAuth(tpl_name string, ctx gonja.Context, gctx echo.Context) error {
	return c.RenderHtml(tpl_name, ctx, gctx, "templates/auth")
}

// Render plain text, mostly used for viewing logs.
func (c *Controller) RenderWithoutLayout(tpl_name string, ctx gonja.Context, gctx echo.Context) error {
	ctx["scriptable_base_url"] = os.Getenv("SCRIPTABLE_URL")
	view, err := gonja.Must(gonja.FromFile("templates/" + tpl_name + ".html")).Execute(ctx)
	if err != nil && utils.LogVerbose() {
		fmt.Println(err)
	}
	return gctx.HTML(http.StatusOK, view)
}

func (c *Controller) GetSessionUser(gctx echo.Context) models.User {
	user_id_interface, err := c.GetSessionValue("user_id", gctx)
	var user models.User

	isValid := false
	if err == nil && user_id_interface != nil {
		switch user_id_interface.(type) {
		case int64:
			isValid = true
			break
		default:
			isValid = false
		}
	}

	if !isValid {
		return user
	}
	models.GetDB().Raw("SELECT id, name, email, verified, team_id FROM users where id = ?", user_id_interface.(int64)).Scan(&user)
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

func (c *Controller) GetSessionValue(key string, e echo.Context) (interface{}, error) {
	sess, err := session.Get("scriptables_session", e)
	if err != nil {
		return nil, err
	}

	fmt.Println(sess.Values)
	value, ok := sess.Values[key]
	if ok {
		return value, nil
	}

	return nil, fmt.Errorf("session key not found")
}

func (c *Controller) SetSessionValues(values map[string]interface{}, e echo.Context) (bool, error) {
	sess, err := session.Get("scriptables_session", e)
	if err != nil {
		return false, err
	}

	for k, v := range values {
		sess.Values[k] = v
	}

	sess.Save(e.Request(), e.Response())
	if err := sess.Save(e.Request(), e.Response()); err != nil {
		fmt.Println(err)
		return false, err
	}

	return true, nil
}

func (c *Controller) DestroySession(e echo.Context) (bool, error) {
	sess, err := session.Get("scriptables_session", e)
	if err != nil {
		return false, err
	}

	sess.Values = nil

	sess.Save(e.Request(), e.Response())
	if err := sess.Save(e.Request(), e.Response()); err != nil {
		return false, err
	}

	return true, nil
}

func (c *Controller) GetFlashMessages(flash_type string, e echo.Context) ([]string, error) {
	sess, err := session.Get("scriptables_session", e)
	if err != nil {
		return nil, err
	}

	flashes := sess.Flashes(flash_type)
	var formatted_messages []string

	for _, v := range flashes {
		formatted_messages = append(formatted_messages, v.(string))
	}

	return formatted_messages, nil
}

func (c *Controller) SetFlashMessages(flashes []string, flash_type string, e echo.Context) (bool, error) {
	sess, err := session.Get("scriptables_session", e)
	if err != nil {
		return false, err
	}

	for _, v := range flashes {
		sess.AddFlash(flash_type, v)
	}

	if err := sess.Save(e.Request(), e.Response()); err != nil {
		return false, err
	}

	return true, nil
}

func (c *Controller) FlashSuccess(e echo.Context, msg string) (bool, error) {
	return c.SetFlashMessages([]string{msg}, models.FLASH_SUCCESS, e)
}

func (c *Controller) FlashError(e echo.Context, msg string) (bool, error) {
	return c.SetFlashMessages([]string{msg}, models.FLASH_ERROR, e)
}
