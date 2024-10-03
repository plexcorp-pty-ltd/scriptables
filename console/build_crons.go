// When you add cron jobs via the interface. This cron will
// sync those jobs to your servers.

package console

import (
	"fmt"

	"gorm.io/gorm"
	"plexcorp.tech/scriptable/models"
)

func setCronStatus(db *gorm.DB, id int64, status string) {
	db.Exec("UPDATE crons SET status=? WHERE id=?", status, id)
}

func processCron(db *gorm.DB, cron *models.Cron) {
	defer func() {

		if r := recover(); r != nil {
			setCronStatus(db, cron.ID, models.STATUS_FAILED)
			fmt.Println("Recovered from serious server cron process failure. Please check the logs for server:", cron.CronName, r)
		}

	}()

	setCronStatus(db, cron.ID, models.STATUS_RUNNING)
	cronFileVirtual := ""
	var ServerCrons []*models.Cron

	server := models.GetServer(cron.ServerID, cron.TeamID)

	db.Table("crons").Where("server_id=? and deleted_at IS NULL", cron.ServerID).Scan(&ServerCrons)

	for _, scron := range ServerCrons {
		cronFileVirtual += fmt.Sprintf("%s %s %s \n", scron.CronExpression, scron.User, scron.Task)
	}

	client, err := models.GetSSHClient(&server, false)
	if err != nil {
		setCronStatus(db, cron.ID, models.STATUS_FAILED)
		models.LogError(cron.ID, "cron", err.Error(),
			"Failed to establish connection to server: "+server.ServerName, server.TeamId)
		return
	}

	cronName := fmt.Sprintf("server_%d_crons", server.ID)
	tmpCronPath := fmt.Sprintf("/home/%s/%s", server.NewSSHUsername, cronName)

	cmd := fmt.Sprintf(`
		#!/bin/bash
		sudo rm -f /etc/cron.d/%s
		sudo systemctl restart cron
		sudo systemctl status cron
	`, cronName)

	if cronFileVirtual != "" {

		err = client.Sftp().WriteFile(tmpCronPath, []byte(cronFileVirtual), 0644)

		if err != nil {
			models.LogError(cron.ID, "cron", err.Error(),
				"Failed to write tmp cron file for server: "+server.ServerName, server.TeamId)
			return
		}
		cmd = fmt.Sprintf(`
			#!/bin/bash
			sudo mv -f %s /etc/cron.d/
			sudo chown -R root:root /etc/cron.d/%s
			sudo chmod 644 /etc/cron.d/%s
			sudo systemctl restart cron
			sudo systemctl status cron
			`, tmpCronPath, cronName, cronName)

	}

	out, err := client.Script(cmd).Output()

	if err != nil {
		models.LogError(cron.ID, "cron", err.Error(),
			"Failed to update crons for: "+server.ServerName, server.TeamId)
		setCronStatus(db, cron.ID, models.STATUS_FAILED)
		return
	}

	models.LogInfo(cron.ID, "cron", string(out)+"\n --- cron list ---\n"+cronFileVirtual,
		"Cronfile update log", server.TeamId)
	models.LogInfo(cron.ServerID, "server", string(out)+"\n --- cron list ---\n"+cronFileVirtual,
		"Cronfile update log", server.TeamId)

	setCronStatus(db, cron.ID, models.STATUS_COMPLETE)
}

func BuildCrons(db *gorm.DB) {
	var crons []*models.Cron

	db.Table("crons").Where("status=?", models.STATUS_QUEUED).Scan(&crons)

	for _, cron := range crons {
		processCron(db, cron)
	}
}
