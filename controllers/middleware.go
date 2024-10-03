package controllers

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"plexcorp.tech/scriptable/models"
)

func AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if strings.Contains(c.Path(), "/users/") ||
			strings.Contains(c.Path(), "/webhooks/") ||
			strings.Contains(c.Path(), "trial-expired") ||
			strings.Contains(c.Path(), "/static/") {
			return next(c)
		}

		allowRegistration, _ := strconv.ParseBool(os.Getenv("ALLOW_REGISTER"))
		if allowRegistration {
			var numUsers int
			models.GetDB().Raw("SELECT count(id) FROM users").Scan(&numUsers)
			if numUsers == 0 {
				return c.Redirect(http.StatusFound, "/users/register")
			}
		}

		controller := Controller{}
		user_id_interface, err := controller.GetSessionValue("user_id", c)
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
			return c.Redirect(http.StatusFound, "/users/login")
		}

		return next(c)
	}
}
