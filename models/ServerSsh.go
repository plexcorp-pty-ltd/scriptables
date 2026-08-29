package models

import (
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"
	"plexcorp.tech/scriptable/sshclient"
	"plexcorp.tech/scriptable/utils"
)

func GetSSHClient(server *ServerDetails, intialConnect bool) (*sshclient.Client, error) {
	if intialConnect {
		return sshclient.DialWithLocalKeys(server.ServerIP+":"+strconv.Itoa(server.SshPort), server.SSHUsername)
	}

	return sshclient.DialWithLocalKeys(server.ServerIP+":"+strconv.Itoa(server.NewSshPort), server.NewSSHUsername)
}

func RunScriptable(db *gorm.DB, entity string, id int64, client *sshclient.Client, cmd string, summary string, logtask bool, teamId int64) (error, string) {

	if utils.LogVerbose() {
		fmt.Println("Running ", summary)
		fmt.Println(cmd)
		fmt.Println(">>>><<<<")
	}

	out, err := client.Script(cmd).Output()

	status := STATUS_COMPLETE
	if err != nil {
		status = STATUS_FAILED
	}
	if logtask {
		step := ScriptableTaskLog{
			EntityID:   id,
			Entity:     entity,
			Task:       summary,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			TaskStatus: status,
			TeamID:     teamId,
		}
		db.Create(&step)
	}

	return err, string(out)
}
