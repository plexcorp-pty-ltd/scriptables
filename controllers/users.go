package controllers

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/noirbizarre/gonja"
	"github.com/pquerna/otp/totp"
	"plexcorp.tech/scriptable/models"
	"plexcorp.tech/scriptable/utils"
)

func (c *Controller) MyProfile(gctx echo.Context) error {

	user := c.GetSessionUser(gctx)
	vars := gonja.Context{
		"title":     "My Profile",
		"user":      user,
		"highlight": "users",
	}

	return c.Render("users/profile", vars, gctx)

}

func (c *Controller) UpdateProfile(gctx echo.Context) error {
	email := strings.Trim(gctx.FormValue("email"), " ")
	name := strings.Trim(gctx.FormValue("name"), " ")
	twoFactor := strings.Trim(gctx.FormValue("two_factor"), " ")
	user := c.GetSessionUser(gctx)
	db := models.GetDB()
	db.Table("users").Where("id", user.ID).First(&user)

	if user.ID == 0 {
		c.FlashError(gctx, "Sorry, failed to fetch your profile. Please try again, if the problem persists. Log out and back in again.")
		return gctx.Redirect(http.StatusFound, "/user/profile")

	}

	if email == "" || name == "" || len(email) < 3 || !strings.Contains(email, "@") || len(name) < 3 {
		c.FlashError(gctx, "Sorry, invalid name or email. Both should be 3 characters or more and email should be a valid email address.")
		return gctx.Redirect(http.StatusFound, "/user/profile")
	}

	if email != user.Email {
		found := 0
		db.Raw("select count(id) as total from users where email=?", email).Scan(&found)
		if found > 0 {
			c.FlashError(gctx, "Sorry, email is already assigned to another user.")
			return gctx.Redirect(http.StatusFound, "/user/profile")
		}
	}

	tf := 0
	if twoFactor == "on" {
		tf = 1
	}

	db.Exec("UPDATE users SET email=?, name=?,two_factor=? WHERE id=?", email, name, tf, user.ID)
	c.FlashSuccess(gctx, "Successfully updated your profile.")
	return gctx.Redirect(http.StatusFound, "/user/profile")
}

func (c *Controller) ShowQrCodePng(gctx echo.Context) error {

	user := c.GetSessionUser(gctx)
	qrcodeBytes, err := utils.ShowQrCode(user.Email, models.GetDB())

	if err != nil {
		c.FlashError(gctx, "Sorry, failed to generate 2Factor QR code. Please try again.")
		return gctx.Redirect(http.StatusFound, "/user/profile")
	}

	gctx.Request().Header.Add("Content-Type", "image/png")
	_, err = gctx.Response().Write(qrcodeBytes)
	return err
}

func (c *Controller) CheckLogin(gctx echo.Context) error {
	vars := gonja.Context{
		"title": "Login",
	}

	vars["errors"] = []string{}

	email := strings.Trim(gctx.FormValue("email"), " ")
	password := strings.Trim(gctx.FormValue("password"), " ")
	if email == "" || password == "" {
		vars["errors"] = []string{"Please enter a valid email address and password."}
	}

	db := models.GetDB()
	var user *models.User
	db.Table("users").Where("email  = ?", email).First(&user)

	if user.TwoFactor == 1 {
		vars := gonja.Context{
			"title":    "Two Factor Login",
			"email":    email,
			"password": password,
		}
		return c.RenderAuth("users/two_factor_confirm", vars, gctx)
	}

	if len(vars["errors"].([]string)) == 0 {
		user, err := models.Authenticate(email, password, gctx.Request().RemoteAddr)
		if err != nil {
			vars["errors"] = []string{err.Error()}
		}

		if user.ID != 0 && user.Email != "" {
			values := make(map[string]interface{})
			values["user_id"] = user.ID
			c.SetSessionValues(values, gctx)
			return gctx.Redirect(http.StatusFound, "/")
		}
	}

	vars["errors"] = []string{"Invalid username or password."}
	return c.RenderAuth("users/login", vars, gctx)
}

func (c *Controller) TwoFactorAuthenticate(gctx echo.Context) error {

	vars := gonja.Context{
		"title": "Two Factor Login",
	}

	db := models.GetDB()

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
	}

	if len(vars["errors"].([]string)) == 0 {
		user, err := models.Authenticate(email, password, gctx.Request().RemoteAddr)
		if err != nil {
			vars["errors"] = []string{err.Error()}
		}

		if user.ID != 0 && user.Email != "" {
			values := make(map[string]interface{})
			values["user_id"] = user.ID
			c.SetSessionValues(values, gctx)
			return gctx.Redirect(http.StatusFound, "/")
		}
	}

	if len(vars["errors"].([]string)) == 0 {
		vars["errors"] = []string{"Invalid 2factor auth code. Please try again."}
	}

	return c.RenderAuth("users/login", vars, gctx)

}

func (c *Controller) LoginView(gctx echo.Context) error {

	vars := gonja.Context{
		"title": "Login",
	}

	return c.RenderAuth("users/login", vars, gctx)

}

func (c *Controller) Logout(gctx echo.Context) error {

	vars := gonja.Context{
		"title": "Login",
	}

	c.DestroySession(gctx)

	return c.RenderAuth("users/login", vars, gctx)
}

func (c *Controller) ForgotPassword(gctx echo.Context) error {

	vars := gonja.Context{
		"title": "Forgot Password",
	}

	if gctx.Request().Method == http.MethodPost {
		email := gctx.FormValue("email")
		isValidEmail := models.IsValidEmail(email, gctx.Request().RemoteAddr)

		if isValidEmail {
			models.SendPasswordResetToken(email, "Password reset request", "forgotpassword")
		}

		vars["successMsg"] = "Please check your email for further instructions."
	}

	return c.RenderAuth("users/forgot", vars, gctx)
}

func (c *Controller) ChangePassword(gctx echo.Context) error {

	token := gctx.Param("token")

	vars := gonja.Context{
		"title": "Change Password",
		"token": token,
	}

	hasErrors := false
	if gctx.Request().Method == http.MethodPost {
		email := gctx.FormValue("email")
		password := gctx.FormValue("password")
		passwordAgain := gctx.FormValue("passwordAgain")
		if password != passwordAgain {
			hasErrors = true
			vars["errors"] = []string{"Ooops!, password and confirm password do not match. Please try again."}
		} else {
			user := models.GetUserByEmailToken(email, token)

			if user.Email == email {
				user.Password, _ = utils.HashPassword(password)
				user.ResetToken = ""

				models.UpdateUserPassword(&user)
				vars["successMsg"] = "Thank you, password successfully reset - you may now login."

			} else {
				hasErrors = true
				vars["errors"] = []string{"Oops! Something went wrong. Please try again - if this failure persists. Please re-request a password reset."}
			}
		}
	}

	if gctx.Request().Method == http.MethodPost && !hasErrors {
		vars["title"] = "Login"
		return c.RenderAuth("users/login", vars, gctx)

	}

	return c.RenderAuth("users/reset", vars, gctx)

}

func (c *Controller) ListUsers(gctx echo.Context) error {
	page, err := strconv.Atoi(gctx.QueryParam("page"))
	sessUser := c.GetSessionUser(gctx)
	if err != nil {
		page = 1
	}

	perPage, err := strconv.Atoi(gctx.QueryParam("perPage"))
	if err != nil {
		perPage = 20
	}

	search := gctx.QueryParam("search")

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

	vars["users"] = models.GetUsersList(page, perPage, search, sessUser.TeamId)

	return c.Render("users/list", vars, gctx)

}

func (c *Controller) HandleUserActionsFormPost(gctx echo.Context) error {
	action := gctx.FormValue("action")
	id, err := strconv.ParseInt(gctx.FormValue("user_id"), 10, 64)
	sessUser := c.GetSessionUser(gctx)
	if err != nil {
		c.FlashError(gctx, "Sorry, failed to read user ID. Please try again.")
	}

	if action == "deactivate" {
		err := models.ToggleUserStatus(id, 0, sessUser.TeamId)
		if err == nil {
			c.FlashSuccess(gctx, "Successfully disabled users access. They won't be able to login.")
		}

	} else if action == "activate" {
		err := models.ToggleUserStatus(id, 1, sessUser.TeamId)
		if err == nil {
			c.FlashSuccess(gctx, "Successfully enabled users access.")
		}
	} else if action == "sendpassword" {
		email := gctx.FormValue("user_email")
		models.SendPasswordResetToken(email, "Password reset request", "forgotpassword")
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
			models.GetDB().Create(&u)
			models.SendPasswordResetToken(email, "Your scriptables account is ready.", "newuser")
			c.FlashSuccess(gctx, "Successfully created new user, they will get an email shortly to setup their login details.")
		}
	}

	return gctx.Redirect(http.StatusFound, "/user/list")
}

func (c *Controller) NewUser(gctx echo.Context) error {
	return c.Render("users/new_user", gonja.Context{
		"serverTypes": models.GetServerTypes(),
		"title":       "Choose server template",
	}, gctx)
}

func (c *Controller) RegisterForm(gctx echo.Context) error {
	vars := gonja.Context{
		"title": "Register for an account",
		"email": "",
		"name":  "",
		"team":  ""}

	allowRegistration, _ := strconv.ParseBool(os.Getenv("ALLOW_REGISTER"))
	if !allowRegistration {
		c.FlashError(gctx, "Registration is currently not allowed. Please enable the ENV flag first.")
		return gctx.Redirect(http.StatusFound, "/denied")
	}

	testEncryption := "testing 12345"
	s := utils.Encrypt(testEncryption)

	if utils.Decrypt(s) != testEncryption {

		vars["errors"] = []string{"Warning: there is a problem with your encryption key. Ensure that it is between 16, 24, 32 characters long. Please update this and restart the docker container."}
	}

	return c.RenderAuth("users/register", vars, gctx)
}

func (c *Controller) RegistrationComplete(gctx echo.Context) error {
	email := gctx.FormValue("email")
	password := gctx.FormValue("password")
	name := gctx.FormValue("name")
	team := gctx.FormValue("team")
	passwordConfirmation := gctx.FormValue("password_confirm")

	allowRegistration, _ := strconv.ParseBool(os.Getenv("ALLOW_REGISTER"))
	if !allowRegistration {
		c.FlashError(gctx, "Registration is currently not allowed. Please enable the ENV flag first.")
		return gctx.Redirect(http.StatusFound, "/denied")
	}

	var errors []string

	vars := gonja.Context{
		"title": "Register for an account",
		"email": email,
		"name":  name,
		"team":  team,
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
		db := models.GetDB()

		var userTeam models.Team
		userTeam.Name = team
		userTeam.CreatedAt = time.Now()
		userTeam.UpdatedAt = time.Now()
		db.Create(&userTeam)
		if userTeam.ID != 0 {
			user.TeamId = userTeam.ID
			db.Create(&user)

			// make SSH key
			if !models.DoesTeamHaveAnSSHKey(user.TeamId) {
				privateKey, publicKey := utils.MakeSSHKey()
				if privateKey != "" && publicKey != "" {
					key := models.SshKey{
						Name:       "Default Scriptables Generated",
						PrivateKey: utils.Encrypt(privateKey),
						PublicKey:  utils.Encrypt(publicKey),
						Passphrase: "",
						CreatedAt:  time.Now(),
						UpdatedAt:  time.Now(),
						TeamId:     user.TeamId,
					}

					db.Create(&key)
				}
			}
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
			c.SetFlashMessages([]string{"Successfully setup your account. You now can login."}, "succes", gctx)
			return gctx.Redirect(http.StatusFound, "/users/login")
		}
	} else {
		vars["errors"] = errors
	}

	return c.RenderAuth("users/register", vars, gctx)
}
