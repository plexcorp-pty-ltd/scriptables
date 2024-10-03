package main

import (
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"plexcorp.tech/scriptable/console"
	"plexcorp.tech/scriptable/controllers"
	"plexcorp.tech/scriptable/models"
)

// This method fires cron jobs found in the console/ directory.
func RunJobs() {

	defer func() {

		if r := recover(); r != nil {
			fmt.Println("Caught and recovered from cron daemon crash:", r)
		}

	}()

	db := models.GetDB()
	console.DeployBranch(db)
	console.BuildServers(db)
	console.BuildSites(db)
	console.BuildCrons(db)

}

func main() {
	location, err := time.LoadLocation(os.Getenv("TZ"))
	if err != nil {
		fmt.Println("Timezone entered is invalid:", err)
		return
	}

	time.Local = location

	go func() {
		for {
			RunJobs()
			time.Sleep(30 * time.Second)
		}
	}()

	mysqlDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		os.Getenv("MYSQL_USER"),
		os.Getenv("MYSQL_PASSWORD"),
		os.Getenv("MYSQL_HOST"),
		os.Getenv("MYSQL_PORT"),
		os.Getenv("MYSQL_DATABASE"),
	)

	os.Setenv("MYSQL_DSN", mysqlDSN)
	router := echo.New()
	router.Static("/static", "./static")

	domain, _ := url.Parse(os.Getenv("SCRIPTABLE_URL"))

	middleware.CSRFWithConfig(middleware.CSRFConfig{
		TokenLookup:    "cookie:_csrf",
		CookiePath:     "/",
		CookieDomain:   domain.Host,
		CookieSecure:   true,
		CookieHTTPOnly: true,
	})

	router.Use(session.Middleware(sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))))

	controller := controllers.Controller{}
	router.GET("/trial-expired", controllers.AuthMiddleware(controller.TrialExpired))
	router.GET("/users/logout", controller.Logout)
	router.GET("/users/login", controller.LoginView)
	router.POST("/users/authenticate", controller.CheckLogin)
	router.POST("/users/2fa/authenticate", controller.TwoFactorAuthenticate)
	router.Any("/users/password/reset/:token", controller.ChangePassword)
	router.Any("/users/password/forgot", controller.ForgotPassword)
	router.POST("/users/register/complete", controller.RegistrationComplete)
	router.GET("/users/register", controller.RegisterForm)

	router.GET("/user/list", controllers.AuthMiddleware(controller.ListUsers))
	router.POST("/user/actions", controllers.AuthMiddleware(controller.HandleUserActionsFormPost))
	router.POST("/user/profile/update", controllers.AuthMiddleware(controller.UpdateProfile))
	router.GET("/user/profile", controllers.AuthMiddleware(controller.MyProfile))
	router.GET("/user/create", controllers.AuthMiddleware(controller.NewUser))

	router.GET("/", controllers.AuthMiddleware(controller.ChooseServerType))

	router.GET("/denied", controller.AccessDenied)

	router.GET("/user/2factor/qrcode", controllers.AuthMiddleware(controller.ShowQrCodePng))
	router.GET("/log/full/server/:id", controllers.AuthMiddleware(controller.ServerLogView))
	router.GET("/logs/server/:id", controllers.AuthMiddleware(controller.ServerLogs))
	router.GET("/log/full/site/:id", controllers.AuthMiddleware(controller.SiteLogView))
	router.GET("/logs/site/:id", controllers.AuthMiddleware(controller.SiteLogs))
	router.GET("/logs/cron/:id", controllers.AuthMiddleware(controller.CronLogs))
	router.GET("/log/full/cron/:id", controllers.AuthMiddleware(controller.CronLogView))

	router.GET("/servers", controllers.AuthMiddleware(controller.Servers))
	router.Any("/server/create/:servertype", controllers.AuthMiddleware(controller.CreateServer))
	router.Any("/server/update/:id", controllers.AuthMiddleware(controller.UpdateServer))
	router.Any("/server/test-ssh/:id", controllers.AuthMiddleware(controller.ShowTestConnectionLoader))
	router.Any("/server/test-ssh-ajax/:id", controllers.AuthMiddleware(controller.TestSSHConnection))
	router.POST("/server/retrybuild", controllers.AuthMiddleware(controller.RetryBuildServer))
	router.GET("/server/firewall/:serverID", controllers.AuthMiddleware(controller.FirewallRules))
	router.Any("/server/firewall-ajax/:serverID", controllers.AuthMiddleware(controller.FirewallRulesAjax))
	router.POST("/server/firewall/delete/rule", controllers.AuthMiddleware(controller.DeleteFirewallRule))
	router.POST("/server/firewall/add/rule", controllers.AuthMiddleware(controller.AddFirewallRule))

	router.GET("/sshkeys", controllers.AuthMiddleware(controller.SshKeys))
	router.GET("/sshkey/create", controllers.AuthMiddleware(controller.CreateSShKey))
	router.GET("/sshkey/edit/:id", controllers.AuthMiddleware(controller.EditSShKey))
	router.POST("/sshkey/save", controllers.AuthMiddleware(controller.SaveSShKey))

	router.GET("/site/deployKey/:id", controllers.AuthMiddleware(controller.CreateSiteDeployKey))
	router.POST("/site/generateDeployKey", controllers.AuthMiddleware(controller.GenerateDeployKey))
	router.POST("/site/deploy/", controllers.AuthMiddleware(controller.DeployBranch))
	router.POST("/site/retrybuild", controllers.AuthMiddleware(controller.RetrySiteBuild))
	router.POST("/site/confirm-deploy", controllers.AuthMiddleware(controller.ConfirmSiteDeploy))
	router.GET("/sites", controllers.AuthMiddleware(controller.Sites))
	router.GET("/site/create", controllers.AuthMiddleware(controller.CreateSite))
	router.POST("/site/save", controllers.AuthMiddleware(controller.SaveSite))

	router.GET("/crons", controllers.AuthMiddleware(controller.Crons))
	router.POST("/cron/save", controllers.AuthMiddleware(controller.SaveCron))
	router.GET("/cron/create", controllers.AuthMiddleware(controller.CreateCron))
	router.GET("/cron/edit/:id", controllers.AuthMiddleware(controller.EditCron))
	router.POST("/cron/update/:id", controllers.AuthMiddleware(controller.UpdateCron))
	router.POST("/cron/disable/", controllers.AuthMiddleware(controller.DisableCron))
	router.POST("/cron/retrybuild", controllers.AuthMiddleware(controller.RetryCronBuild))

	router.GET("/systemd/services/:id/list", controllers.AuthMiddleware(controller.ListServices))

	router.GET("/guide", controller.ShowGuide)

	router.GET("/webhooks/deploy/:sid/:token", controllers.AuthMiddleware(controller.DeployWebhookSite))

	router.Start(os.Getenv("SCRIPTABLES_SERVER_DSN_HOST") + ":" + os.Getenv("SCRIPTABLES_SERVER_DSN_PORT"))

}
