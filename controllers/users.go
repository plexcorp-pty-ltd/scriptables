package controllers

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/noirbizarre/gonja"
	"github.com/pquerna/otp/totp"
	"plexcorp.tech/scriptable/models"
	"plexcorp.tech/scriptable/utils"
)

func (c *Controller) MyProfile(gctx *gin.Context) {

	session := sessions.Default(gctx)
	userID := session.Get("user_id")

	var user models.User
	c.GetDB(gctx).Table("users").Where("id=?", userID).First(&user)

	vars := gonja.Context{
		"title":     "My Profile",
		"user":      user,
		"highlight": "users",
	}

	c.Render("users/profile", vars, gctx)

}

func (c *Controller) UpdateProfile(gctx *gin.Context) {
	email := strings.Trim(gctx.FormValue("email"), " ")
	name := strings.Trim(gctx.FormValue("name"), " ")
	twoFactor := strings.Trim(gctx.FormValue("two_factor"), " ")

	session := sessions.Default(gctx)
	userID := session.Get("user_id")

	var user models.User
	c.GetDB(gctx).Table("users").Where("id", userID).First(&user)

	if user.ID == 0 {
		c.FlashError(gctx, "Sorry, failed to fetch your profile. Please try again, if the problem persists. Log out and back in again.")
		gctx.Redirect(http.StatusFound, "/user/profile")
		return
	}

	if email == "" || name == "" || len(email) < 3 || !strings.Contains(email, "@") || len(name) < 3 {
		c.FlashError(gctx, "Sorry, invalid name or email. Both should be 3 characters or more and email should be a valid email address.")
		gctx.Redirect(http.StatusFound, "/user/profile")
		return
	}

	db := c.GetDB(gctx)

	if email != user.Email {
		found := 0
		db.Raw("select count(id) as total from users where email=?", email).Scan(&found)
		if found > 0 {
			c.FlashError(gctx, "Sorry, email is already assigned to another user.")
			gctx.Redirect(http.StatusFound, "/user/profile")
			return
		}
	}

	tf := 0
	if twoFactor == "on" {
		tf = 1
	}

	db.Exec("UPDATE users SET email=?, name=?,two_factor=? WHERE id=?", email, name, tf, user.ID)
	c.FlashSuccess(gctx, "Successfully updated your profile.")
	gctx.Redirect(http.StatusFound, "/user/profile")
}

func (c *Controller) ShowQrCodePng(gctx *gin.Context) {

	session := sessions.Default(gctx)
	userID := session.Get("user_id")

	var user models.User
	c.GetDB(gctx).Where("id=?", userID).Find(&user)
	qrcodeBytes, err := utils.ShowQrCode(user.Email, c.GetDB(gctx))

	if err != nil {
		c.FlashError(gctx, "Sorry, failed to generate 2Factor QR code. Please try again.")
		gctx.Redirect(http.StatusFound, "/user/profile")
		return
	}

	gctx.Header("Content-Type", "image/png")
	gctx.Writer.Write(qrcodeBytes)

}

func (c *Controller) CheckLogin(gctx *gin.Context) {

	if !c.TestCSRFToken(gctx) {
		c.FlashError(gctx, "Sorry, your session has expired. Please try refreshing this page.")
		gctx.Redirect(http.StatusFound, "/users/login")
		return
	}

	vars := gonja.Context{
		"title": "Login",
	}

	vars["errors"] = []string{}

	email := strings.Trim(gctx.FormValue("email"), " ")
	password := strings.Trim(gctx.FormValue("password"), " ")
	if email == "" || password == "" {
		vars["errors"] = []string{"Please enter a valid email address and password."}
	}

	db := c.GetDB(gctx)
	var user *models.User
	db.Table("users").Where("email  = ?", email).First(&user)

	if user.TwoFactor == 1 {
		vars := gonja.Context{
			"title":    "Two Factor Login",
			"email":    email,
			"password": password,
		}
		c.RenderAuth("users/two_factor_confirm", vars, gctx)
		return
	}

	if len(vars["errors"].([]string)) == 0 {
		user, err := models.Authenticate(db, email, password, gctx.Request.RemoteAddr)
		if err != nil {
			vars["errors"] = []string{err.Error()}
		}

		if user.ID != 0 && user.Email != "" {
			session := sessions.Default(gctx)
			session.Set("user_id", user.ID)
			session.Save()
			gctx.Redirect(http.StatusFound, "/")
			return
		}
	}

	vars["errors"] = []string{"Invalid username or password."}
	c.RenderAuth("users/login", vars, gctx)

}

func (c *Controller) TwoFactorAuthenticate(gctx *gin.Context) {

	vars := gonja.Context{
		"title": "Two Factor Login",
	}

	if !c.TestCSRFToken(gctx) {
		c.FlashError(gctx, "Sorry, your session has expired. Please try refreshing this page.")
		gctx.Redirect(http.StatusFound, "/users/login")
		return
	}

	db := c.GetDB(gctx)

	vars["errors"] = []string{}

	email := strings.Trim(gctx.FormValue("email"), " ")
	password := strings.Trim(gctx.FormValue("password"), " ")
	twoFactorCode := strings.Trim(gctx.FormValue("two_factor_code"), " ")

	if twoFactorCode == "" {
		vars["errors"] = []string{"Invalid two factor code entered."}
	} else if email == "" || password == "" {
		vars["errors"] = []string{"Please enter a valid email address and password."}
	}

	var user *models.User
	db.Table("users").Where("email  = ?", email).First(&user)

	valid := totp.Validate(twoFactorCode, utils.Decrypt(user.TwoFactorCode))
	if !valid {

		vars := gonja.Context{
			"title":    "Two Factor Login",
			"email":    email,
			"password": password,
			"errors":   []string{"Invalid two factor auth code entered. Please try again."},
		}
		c.RenderAuth("users/two_factor_confirm", vars, gctx)
		return
	}

	if len(vars["errors"].([]string)) == 0 {
		user, err := models.Authenticate(db, email, password, gctx.Request.RemoteAddr)
		if err != nil {
			vars["errors"] = []string{err.Error()}
		}

		if user.ID != 0 && user.Email != "" {
			session := sessions.Default(gctx)
			session.Set("user_id", user.ID)
			session.Save()
			gctx.Redirect(http.StatusFound, "/")
			return
		}
	}

	if len(vars["errors"].([]string)) == 0 {
		vars["errors"] = []string{"Invalid 2factor auth code. Please try again."}
	}

	c.RenderAuth("users/login", vars, gctx)

}

func (c *Controller) LoginView(gctx *gin.Context) {

	vars := gonja.Context{
		"title": "Login",
	}

	c.RenderAuth("users/login", vars, gctx)

}

func (c *Controller) Logout(gctx *gin.Context) {

	vars := gonja.Context{
		"title": "Login",
	}

	session := sessions.Default(gctx)
	session.Clear()
	session.Save()

	c.RenderAuth("users/login", vars, gctx)
}

func (c *Controller) ForgotPassword(gctx *gin.Context) {

	vars := gonja.Context{
		"title": "Forgot Password",
	}

	if gctx.Request.Method == http.MethodPost {
		if !c.TestCSRFToken(gctx) {
			c.FlashError(gctx, "Sorry, your session has expired. Please try refreshing this page.")
			gctx.Redirect(http.StatusFound, "/users/login")
			return
		}
		email := gctx.FormValue("email")
		isValidEmail := models.IsValidEmail(c.GetDB(gctx), email, gctx.Request.RemoteAddr)

		if isValidEmail {
			models.SendPasswordResetToken(c.GetDB(gctx), email, "Password reset request", "forgotpassword")
		}

		vars["successMsg"] = "Please check your email for further instructions."
	}

	c.RenderAuth("users/forgot", vars, gctx)

}

func (c *Controller) ChangePassword(gctx *gin.Context) {

	token := gctx.Param("token")

	vars := gonja.Context{
		"title": "Change Password",
		"token": token,
	}

	hasErrors := false
	if gctx.Request.Method == http.MethodPost {
		if !c.TestCSRFToken(gctx) {
			c.FlashError(gctx, "Sorry, your session has expired. Please try refreshing this page.")
			gctx.Redirect(http.StatusFound, "/users/login")
			return
		}
		email := gctx.FormValue("email")
		password := gctx.FormValue("password")
		passwordAgain := gctx.FormValue("passwordAgain")
		if password != passwordAgain {
			hasErrors = true
			vars["errors"] = []string{"Ooops!, password and confirm password do not match. Please try again."}
		} else {
			user := models.GetUserByEmailToken(c.GetDB(gctx), email, token)

			if user.Email == email {
				user.Password, _ = utils.HashPassword(password)
				user.ResetToken = ""

				models.UpdateUserPassword(c.GetDB(gctx), &user)
				vars["successMsg"] = "Thank you, password successfully reset - you may now login."

			} else {
				hasErrors = true
				vars["errors"] = []string{"Oops! Something went wrong. Please try again - if this failure persists. Please re-request a password reset."}
			}
		}
	}

	if gctx.Request.Method == http.MethodPost && !hasErrors {
		vars["title"] = "Login"
		c.RenderAuth("users/login", vars, gctx)
		return

	} else {
		c.RenderAuth("users/reset", vars, gctx)
		return

	}

}

func (c *Controller) ListUsers(gctx *gin.Context) {
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

	searchQuery := ""

	if search != "" {
		searchQuery = "&search=" + searchQuery
	}

	vars := gonja.Context{
		"highlight":   "users",
		"title":       "Users",
		"nextPage":    page + 1,
		"prevPage":    page - 1,
		"searchQuery": searchQuery,
		"search":      search,
		"addBtn":      "<a href=\"/users/create\" data-toggle=\"modal\" data-target=\"#newUserModal\" class=\"btn-sm btn-success\" style=\"vertical-align:middle;\">ADD User</a>",
	}

	vars["users"] = models.GetUsersList(c.GetDB(gctx), page, perPage, search, sessUser.TeamId)

	c.Render("users/list", vars, gctx)

}

func (c *Controller) HandleUserActionsFormPost(gctx *gin.Context) {
	action := gctx.FormValue("action")
	id, err := strconv.ParseInt(gctx.FormValue("user_id"), 10, 64)
	sessUser := c.GetSessionUser(gctx)
	if err != nil {
		c.FlashError(gctx, "Sorry, failed to read user ID. Please try again.")
	}

	if action == "deactivate" {
		err := models.ToggleUserStatus(c.GetDB(gctx), id, 0, sessUser.TeamId)
		if err == nil {
			c.FlashSuccess(gctx, "Successfully disabled users access. They won't be able to login.")
		}

	} else if action == "activate" {
		err := models.ToggleUserStatus(c.GetDB(gctx), id, 1, sessUser.TeamId)
		if err == nil {
			c.FlashSuccess(gctx, "Successfully enabled users access.")
		}
	} else if action == "sendpassword" {
		email := gctx.FormValue("user_email")
		models.SendPasswordResetToken(c.GetDB(gctx), email, "Password reset request", "forgotpassword")
		c.FlashSuccess(gctx, "Successfully sent user a password reset mail.")
	} else if action == "newuser" {
		email := strings.Trim(gctx.FormValue("email"), " ")
		name := strings.Trim(gctx.FormValue("name"), " ")

		if email == "" || name == "" || !strings.Contains(email, "@") || len(name) < 3 {
			c.FlashError(gctx, "Email and name are both required.")
		} else {
			password, _ := utils.HashPassword(utils.GenPassword())
			u := models.User{Email: email, Name: name, CreatedAt: time.Now(), UpdatedAt: time.Now(),
				Verified: 1, Password: password, TeamId: sessUser.TeamId}
			c.GetDB(gctx).Create(&u)
			models.SendPasswordResetToken(c.GetDB(gctx), email, "Your scriptables account is ready.", "newuser")
			c.FlashSuccess(gctx, "Successfully created new user, they will get an email shortly to setup their login details.")
		}
	}

	gctx.Redirect(http.StatusFound, "/user/list")
}

func (c *Controller) NewUser(gctx *gin.Context) {
	c.Render("users/new_user", gonja.Context{
		"serverTypes": models.GetServerTypes(),
		"title":       "Choose server template",
	}, gctx)
}

func (c *Controller) RegisterForm(gctx *gin.Context) {
	vars := gonja.Context{
		"title": "Register for an account",
		"email": "",
		"name":  "",
		"team":  ""}

	allowRegistration, _ := strconv.ParseBool(os.Getenv("ALLOW_REGISTER"))
	if !allowRegistration {
		c.FlashError(gctx, "Registration is currently not allowed. Please enable the ENV flag first.")
		gctx.Redirect(http.StatusFound, "/denied")
		return
	}

	testEncryption := "testing 12345"
	s := utils.Encrypt(testEncryption)

	if utils.Decrypt(s) != testEncryption {

		vars["errors"] = []string{"Warning: there is a problem with your encryption key. Ensure that it is between 16, 24, 32 characters long. Please update this and restart the docker container."}
	}

	c.RenderAuth("users/register", vars, gctx)
}

func (c *Controller) RegistrationComplete(gctx *gin.Context) {
	email := gctx.FormValue("email")
	password := gctx.FormValue("password")
	name := gctx.FormValue("name")
	team := gctx.FormValue("team")
	passwordConfirmation := gctx.FormValue("password_confirm")

	allowRegistration, _ := strconv.ParseBool(os.Getenv("ALLOW_REGISTER"))
	if !allowRegistration {
		c.FlashError(gctx, "Registration is currently not allowed. Please enable the ENV flag first.")
		gctx.Redirect(http.StatusFound, "/denied")
		return
	}

	var errors []string

	vars := gonja.Context{
		"title": "Register for an account",
		"email": email,
		"name":  name,
		"team":  team,
	}

	if !c.TestCSRFToken(gctx) {
		c.FlashError(gctx, "Sorry, your session has expired. Please try refreshing this page.")
		gctx.Redirect(http.StatusFound, "/users/login")
		return
	}

	if password != passwordConfirmation {
		errors = append(errors, "Password and password confirmation do not match.")
	}

	if team == "" {
		errors = append(errors, "Please specify a team name, this can be your company name.")
	}

	if len(password) < 6 {
		errors = append(errors, "Password must contain at least 6 characters.")
	}
	if len(email) < 5 || !strings.Contains(email, "@") {
		errors = append(errors, "Email is not a valid email address")
	}

	password, err := utils.HashPassword(password)
	if err != nil {

		errors = append(errors, "Failed to hash password: ", err.Error(), ". Please try again.")
	}

	if len(errors) == 0 {
		var user models.User
		user.Email = email
		user.CreatedAt = time.Now()
		user.Verified = 1
		user.Name = name
		user.Password = password
		db := c.GetDB(gctx)

		var userTeam models.Team
		userTeam.Name = team
		userTeam.CreatedAt = time.Now()
		userTeam.UpdatedAt = time.Now()
		db.Create(&userTeam)
		if userTeam.ID != 0 {
			user.TeamId = userTeam.ID
			db.Create(&user)
		}
		if user.ID == 0 {
			vars["errors"] = []string{"Sorry, something went wrong. Please try again."}
		} else {
			vars := gonja.Context{
				"subject": "Welcome to Scriptables!",
				"name":    user.Name,
				"email":   user.Email,
			}

			utils.SendEmail("Welcome to Scriptables!", "", []string{user.Email}, vars, "welcome")
			c.FlashSuccess(gctx, "Successfully setup your account. You now can login.")
			gctx.Redirect(http.StatusFound, "/users/login")
			return
		}
	} else {
		vars["errors"] = errors
	}

	c.RenderAuth("users/register", vars, gctx)
}
