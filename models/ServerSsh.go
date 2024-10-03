package models

import (
	"fmt"
	"strconv"
	"time"

	"plexcorp.tech/scriptable/sshclient"
	"plexcorp.tech/scriptable/utils"
)

func GetSSHClient(server *ServerWithSShKey, intialConnect bool) (*sshclient.Client, error) {
	pk := utils.Decrypt(server.PrivateKey)
	pass := ""

	if server.Passphrase != "" {
		pass = utils.Decrypt(server.Passphrase)
	}

	var client *sshclient.Client
	var err error

	if intialConnect {
		client, err = sshclient.DialWithKey(server.ServerIP+":"+strconv.Itoa(server.SshPort), server.SSHUsername, pk, pass)

	} else {
		client, err = sshclient.DialWithKey(server.ServerIP+":"+strconv.Itoa(server.NewSshPort), server.NewSSHUsername, pk, pass)
	}
	return client, err
}

func RunScriptable(entity string, id int64, client *sshclient.Client, cmd string, summary string, logtask bool, teamId int64) (error, string) {

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
		GetDB().Create(&step)
	}

	return err, string(out)
}
