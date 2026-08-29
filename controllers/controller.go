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
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/noirbizarre/gonja"
	"gorm.io/gorm"
	"plexcorp.tech/scriptable/models"
	"plexcorp.tech/scriptable/utils"
)

// We not using the default HTML templating engine that GIN uses. Jinja is a bit easier to work with
// and is cleaner generally. Therefore - for this to work in GIN, a custom struct and rendering methods
// where needed. See RenderHtml - for how this is used.
type JinjaRender string

func (n JinjaRender) Render(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err := w.Write([]byte(n))
	return err
}

func (n JinjaRender) WriteContentType(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
}

type Controller struct {
}

func (c *Controller) GetDB(gctx *gin.Context) *gorm.DB {
	return gctx.MustGet("db").(*gorm.DB)
}

func (c *Controller) RenderHtml(tpl_name string, ctx gonja.Context, gctx *gin.Context, layoutTpl string) {

	session := sessions.Default(gctx)
	flashes := session.Flashes("success")
	if len(flashes) == 1 {
		ctx["successMsg"] = flashes[0].(string)
	}

	errorMessage := session.Flashes("error")
	if len(errorMessage) == 1 {
		errors, _ := ctx["errrors"].([]string)
		ctx["errors"] = append(errors, errorMessage[0].(string))
	}

	if len(errorMessage) > 0 || len(flashes) > 0 {
		session.Save()
	}

	_, ok := ctx["highlight"]
	if !ok {
		ctx["highlight"] = ""
	}

	ctx["scriptable_base_url"] = os.Getenv("SCRIPTABLE_URL")
	ctx["current_year"] = time.Now().Year()
	ctx["STATUS_QUEUED"] = models.STATUS_QUEUED
	ctx["STATUS_RUNNING"] = models.STATUS_RUNNING
	ctx["STATUS_FAILED"] = models.STATUS_FAILED
	ctx["STATUS_CONNECTING"] = models.STATUS_CONNECTING
	ctx["STATUS_COMPLETE"] = models.STATUS_COMPLETE

	ctx["_csrf_token"] = c.SetAndGetCSRFToken(gctx)

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

	gctx.Render(http.StatusOK, JinjaRender(tpl))
}

// Renders templates to authenticated users only.
func (c *Controller) Render(tpl_name string, ctx gonja.Context, gctx *gin.Context) {
	c.RenderHtml(tpl_name, ctx, gctx, "templates/master")
}

// This Render method renders public templates where authentication is not required.
func (c *Controller) RenderAuth(tpl_name string, ctx gonja.Context, gctx *gin.Context) {
	c.RenderHtml(tpl_name, ctx, gctx, "templates/auth")
}

// Render plain text, mostly used for viewing logs.
func (c *Controller) RenderWithoutLayout(tpl_name string, ctx gonja.Context, gctx *gin.Context) {
	ctx["scriptable_base_url"] = os.Getenv("SCRIPTABLE_URL")
	view, err := gonja.Must(gonja.FromFile("templates/" + tpl_name + ".jinja")).Execute(ctx)
	if err != nil && utils.LogVerbose() {
		fmt.Println(err)
	}
	gctx.Render(http.StatusOK, JinjaRender(view))
}

func (c *Controller) FlashSuccess(gctx *gin.Context, msg string) {
	session := sessions.Default(gctx)
	session.AddFlash(msg, "success")
	session.Save()
}

func isTimestampWithin5Minutes(timestampStr string) bool {
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil && utils.LogVerbose() {
		fmt.Println("Error parsing timestamp:", err)
		return true
	}

	timestampTime := time.Unix(timestamp, 0)
	timeDiff := time.Since(timestampTime)
	return timeDiff >= 2*time.Minute
}

func (c *Controller) SetAndGetCSRFToken(gctx *gin.Context) string {
	session := sessions.Default(gctx)
	sessSet := session.Get("csrfToken")
	setToken := true
	token := ""

	if sessSet != nil && sessSet.(string) != "" {

		token = sessSet.(string)
		tStamp := strings.Split(sessSet.(string), "|")
		if len(tStamp) == 2 {
			tStampStr := tStamp[1]
			setToken = isTimestampWithin5Minutes(tStampStr)
		}
	}

	if setToken {
		token := fmt.Sprintf("%s|%d", uuid.New().String(), time.Now().Unix())
		session.Set("csrfToken", token)
		session.Save()
	}

	return token
}

func (c *Controller) TestCSRFToken(gctx *gin.Context) bool {
	token := gctx.PostForm("_csrf_token")

	session := sessions.Default(gctx)
	return session.Get("csrfToken").(string) == token
}

func (c *Controller) FlashError(gctx *gin.Context, msg string) {
	session := sessions.Default(gctx)
	session.AddFlash(msg, "error")
	session.Save()
}

func (c *Controller) GetSessionUser(gctx *gin.Context) models.User {
	session := sessions.Default(gctx)
	userId := session.Get("user_id").(int64)
	var user models.User
	c.GetDB(gctx).Raw("SELECT id, name, email, verified, team_id FROM users where id = ?", userId).Scan(&user)
	return user
}

func (c *Controller) AccessDenied(gctx *gin.Context) {
	c.Render("general/permission", gonja.Context{
		"highlight": "",
	}, gctx)
}

func (c *Controller) TrialExpired(gctx *gin.Context) {
	c.Render("general/trial_expired", gonja.Context{
		"highlight": "",
	}, gctx)
}
