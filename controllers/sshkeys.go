package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/noirbizarre/gonja"
	"plexcorp.tech/scriptable/models"
	"plexcorp.tech/scriptable/utils"
)

func (c *Controller) CreateSShKey(gctx *gin.Context) {

	c.Render("sshkeys/form", gonja.Context{
		"title":     "Setup an SSH key",
		"sshkey_id": 0,
		"highlight": "sshkeys",
	}, gctx)

}

func (c *Controller) EditSShKey(gctx *gin.Context) {

	sshkeyId, err := strconv.ParseInt(gctx.Param("id"), 10, 64)
	sessUser := c.GetSessionUser(gctx)

	if err != nil {
		c.FlashError(gctx, "Sorry, invalid SSH key ID.")
		gctx.Redirect(http.StatusFound, "/sshkeys")
		return
	}

	var sshkey models.SshKey
	c.GetDB(gctx).Where("id = ? and team_id=?", sshkeyId, sessUser.TeamId).First(&sshkey)

	if sshkey.ID == 0 {
		c.FlashError(gctx, "Sorry, invalid SSH key ID.")
		gctx.Redirect(http.StatusFound, "/sshkeys")
		return
	}

	pass := sshkey.Passphrase
	if pass != "" {
		pass = utils.Decrypt(sshkey.Passphrase)
	}

	c.Render("sshkeys/form", gonja.Context{
		"title":       "Update SSH key: " + sshkey.Name,
		"name":        sshkey.Name,
		"sshkey_id":   sshkeyId,
		"private_key": utils.Decrypt(sshkey.PrivateKey),
		"public_key":  utils.Decrypt(sshkey.PublicKey),
		"passphrase":  pass,
		"highlight":   "sshkeys",
	}, gctx)

}

func (c *Controller) SaveSShKey(gctx *gin.Context) {
	privateKey := gctx.FormValue("private_key")
	publicKey := gctx.FormValue("public_key")
	passphrase := gctx.FormValue("passphrase")
	sshkeyId, _ := strconv.ParseInt(gctx.FormValue("sshkey_id"), 10, 64)

	name := gctx.FormValue("name")

	ctx := gonja.Context{

		"title":       "Setup an SSH key",
		"private_key": privateKey,
		"public_key":  publicKey,
		"passphrase":  passphrase,
		"name":        name,
		"sshkey_id":   sshkeyId,
		"highlight":   "sshkeys",
	}

	saveForm := true
	if privateKey == "" || publicKey == "" || len(privateKey) < 10 || len(publicKey) < 10 {
		ctx["errors"] = []string{"Invalid private or public keys entered."}
		saveForm = false
	}

	if name == "" || len(name) < 3 {
		ctx["errors"] = []string{"Please specify a name for this key of at least 3 characters long."}
		saveForm = false
	}

	if saveForm {
		db := c.GetDB(gctx)
		sessUser := c.GetSessionUser(gctx)
		key := models.SshKey{
			Name:       name,
			PrivateKey: utils.Encrypt(privateKey),
			PublicKey:  utils.Encrypt(publicKey),
			Passphrase: utils.Encrypt(passphrase),
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			TeamId:     sessUser.TeamId,
		}

		if sshkeyId == 0 {
			db.Create(&key)
		} else {
			key.ID = sshkeyId
			db.Save(&key)
		}

		c.FlashSuccess(gctx, "Successfully saved ssh key.")
		gctx.Redirect(http.StatusFound, "/sshkeys")
		return
	}

	c.Render("sshkeys/create", ctx, gctx)

}

func (c *Controller) SshKeys(gctx *gin.Context) {
	page, err := strconv.Atoi(gctx.Query("page"))
	if err != nil {
		page = 1
	}

	perPage, err := strconv.Atoi(gctx.Query("perPage"))
	if err != nil {
		perPage = 20
	}

	search := gctx.Query("search")
	sessUser := c.GetSessionUser(gctx)
	keys := models.GetSshKeysList(c.GetDB(gctx), page, perPage, search, sessUser.TeamId)
	searchQuery := ""

	if search != "" {
		searchQuery = "&search=" + searchQuery
	}

	vars := gonja.Context{
		"title":       "SSH Keys",
		"keys":        keys,
		"nextPage":    page + 1,
		"prevPage":    page - 1,
		"searchQuery": searchQuery,
		"search":      search,
		"addBtn":      "<a href=\"/sshkey/create\" class=\"btn-sm btn-success\" style=\"vertical-align:middle;\">ADD Key</a>",
		"highlight":   "sshkeys",
	}

	c.Render("sshkeys/list", vars, gctx)

}
