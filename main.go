package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
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

	db, err := models.GetAppDB()

	if err == nil {
		sqlDB, err := db.DB()

		if err == nil {
			sqlDB.SetMaxIdleConns(10)
			sqlDB.SetMaxOpenConns(100)
			sqlDB.SetConnMaxLifetime(time.Minute * 30)
			defer sqlDB.Close()
		}
	}

	if err != nil {
		fmt.Println(err)
		return
	}

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
	router.Use(session.Middleware(sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))))
	router.Static("/static", http.Dir("./static"))

	controller := controllers.Controller{}
	router.GET("/trial-expired", controller.TrialExpired)
	router.GET("/users/logout", controller.Logout)
	router.GET("/users/login", controller.LoginView)
	router.POST("/users/authenticate", controller.CheckLogin)
	router.POST("/users/2fa/authenticate", controller.TwoFactorAuthenticate)
	router.Any("/users/password/reset/:token", controller.ChangePassword)
	router.Any("/users/password/forgot", controller.ForgotPassword)
	router.POST("/users/register/complete", controller.RegistrationComplete)
	router.GET("/users/register", controller.RegisterForm)

	router.GET("/user/list", controller.ListUsers)
	router.POST("/user/actions", controller.HandleUserActionsFormPost)
	router.POST("/user/profile/update", controller.UpdateProfile)
	router.GET("/user/profile", controller.MyProfile)
	router.GET("/user/create", controller.NewUser)

	router.GET("/", controller.ChooseServerType)

	router.GET("/denied", controller.AccessDenied)
	router.GET("/user/2factor/qrcode", controller.ShowQrCodePng)
	router.GET("/log/full/server/:id", controller.ServerLogView)
	router.GET("/logs/server/:id", controller.ServerLogs)
	router.GET("/log/full/site/:id", controller.SiteLogView)
	router.GET("/logs/site/:id", controller.SiteLogs)
	router.GET("/logs/cron/:id", controller.CronLogs)
	router.GET("/log/full/cron/:id", controller.CronLogView)

	router.GET("/servers", controller.Servers)
	router.Any("/server/create/:servertype", controller.CreateServer)
	router.Any("/server/update/:id", controller.UpdateServer)
	router.Any("/server/test-ssh/:id", controller.ShowTestConnectionLoader)
	router.Any("/server/test-ssh-ajax/:id", controller.TestSSHConnection)
	router.POST("/server/retrybuild", controller.RetryBuildServer)
	router.GET("/server/firewall/:serverID", controller.FirewallRules)
	router.Any("/server/firewall-ajax/:serverID", controller.FirewallRulesAjax)
	router.POST("/server/firewall/delete/rule", controller.DeleteFirewallRule)
	router.POST("/server/firewall/add/rule", controller.AddFirewallRule)

	router.GET("/sshkeys", controller.SshKeys)
	router.GET("/sshkey/create", controller.CreateSShKey)
	router.GET("/sshkey/edit/:id", controller.EditSShKey)
	router.POST("/sshkey/save", controller.SaveSShKey)

	router.GET("/site/deployKey/:id", controller.CreateSiteDeployKey)
	router.POST("/site/generateDeployKey", controller.GenerateDeployKey)
	router.POST("/site/deploy/", controller.DeployBranch)
	router.POST("/site/retrybuild", controller.RetrySiteBuild)
	router.POST("/site/confirm-deploy", controller.ConfirmSiteDeploy)
	router.GET("/sites", controller.Sites)
	router.GET("/site/create", controller.CreateSite)
	router.POST("/site/save", controller.SaveSite)

	router.GET("/crons", controller.Crons)
	router.GET("/cron/create", controller.CreateCron)
	router.POST("/cron/save", controller.SaveCron)
	router.GET("/cron/edit/:id", controller.EditCron)
	router.POST("/cron/update/:id", controller.UpdateCron)
	router.POST("/cron/disable/", controller.DisableCron)
	router.POST("/cron/retrybuild", controller.RetryCronBuild)

	router.GET("/systemd/services/:id/list", controller.ListServices)

	router.GET("/guide", controller.ShowGuide)

	router.GET("/webhooks/deploy/:sid/:token", controller.DeployWebhookSite)

	router.StartAutoTLS(os.Getenv("SCRIPTABLES_SERVER_DSN_HOST") + ":" + os.Getenv("SCRIPTABLES_SERVER_DSN_PORT"))

}
