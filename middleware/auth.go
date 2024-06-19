package middleware

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/labstack/echo/v4"
)

func AuthMiddleware() echo.MiddlewareFunc {
	return func(c echo.Context) {
		if strings.Contains(c.Path(), "/users/") ||
			strings.Contains(c.Path(), "/webhooks/") ||
			strings.Contains(c.Path(), "trial-expired") {
			c.Next()
			return
		}

		allowRegistration, _ := strconv.ParseBool(os.Getenv("ALLOW_REGISTER"))
		if allowRegistration {
			GetDB().Raw("SELECT count(id) FROM users").Scan(&numUsers)
			if numUsers == 0 {
				c.Redirect(http.StatusFound, "/users/register")
				c.Abort()
				return
			}
		}

		sess, _ := sessions.Get("session", c)
		userID, exists := session["userID"]
		if !exists {
			c.Redirect(http.StatusFound, "/users/login")
			c.Abort()
			return
		}

		c.Next()
	}
}
