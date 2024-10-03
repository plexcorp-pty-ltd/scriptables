package utils

import (
	"context"
	"fmt"
	"net/smtp"
	"os"
	"time"

	"github.com/noirbizarre/gonja"
)

func SendEmail(subject string, from string, recipients []string, vars gonja.Context, template string) {

	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUsername := os.Getenv("SMTP_USERNAME")
	smtpPassword := os.Getenv("SMTP_PASSWORD")

	if from == "" {
		from = os.Getenv("SMTP_FROM_EMAIL")
	}

	if smtpUsername == "xxxx" {
		fmt.Println("Oops! SMTP mail is not configured. Skipping sending this email.")
		return
	}

	defer func() {

		if r := recover(); r != nil {
			fmt.Println("Caught and recovered from mail sender crash:", r, "Subject: ", subject, "Recipients: ", recipients)
		}

	}()
	vars["scriptable_base_url"] = os.Getenv("SCRIPTABLE_URL")

	view, err := gonja.Must(gonja.FromFile("templates/emails/" + template + ".html")).Execute(vars)

	if err != nil {
		fmt.Println(err)
	}

	vars["view"] = view
	master := gonja.Must(gonja.FromFile("templates/emails/master.html"))
	tpl, err := master.Execute(vars)

	if err != nil {
		fmt.Println(err)
	}

	message := "From: " + from + "\n"
	message += "To: " + recipients[0] + "\n"
	message += fmt.Sprintf("Subject: %s\r\n", subject)
	message += "MIME-version: 1.0;\r\n"
	message += "Content-Type: text/html; charset=\"UTF-8\";\r\n"
	message += "Content-Transfer-Encoding: 7bit;\r\n"
	message += "\r\n"
	message += tpl

	if from == "" {
		from = os.Getenv("SMTP_FROM")
	}

	auth := smtp.PlainAuth("", smtpUsername, smtpPassword, smtpHost)
	address := fmt.Sprintf("%s:%s", smtpHost, smtpPort)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	done := make(chan bool)
	go func() {
		err := smtp.SendMail(address, auth, from, recipients, []byte(message))
		if err != nil {
			fmt.Println(err)
		}
		done <- true
	}()

	select {
	case <-done:
	case <-ctx.Done():
		fmt.Println("Mail send timed out", "Subject: ", subject, "Recipients: ", recipients)
	}

}
