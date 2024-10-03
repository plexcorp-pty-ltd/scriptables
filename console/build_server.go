// This Job manages all the tasks relating to setting up and configuring your servers.
package console

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
	"plexcorp.tech/scriptable/models"
	"plexcorp.tech/scriptable/parsers"
	"plexcorp.tech/scriptable/sshclient"
	"plexcorp.tech/scriptable/utils"
)

func runServerBuild(db *gorm.DB, server *models.ServerWithSShKey, scriptables []string, wg *sync.WaitGroup) error {
	db.Model(&models.Server{}).Where("id", server.ID).Update("status", models.STATUS_RUNNING)

	defer func() {
		wg.Done()
		if r := recover(); r != nil {
			db.Model(&models.Server{}).Where("id", server.ID).Update("status", "Failed")
			fmt.Println("Recovered from serious server build failure. Please check the logs for server:", server.ServerName, r)
		}

	}()

	pk := utils.Decrypt(server.PrivateKey)
	pass := ""

	if server.Passphrase != "" {
		pass = utils.Decrypt(server.Passphrase)
	}

	var securityStepComplete int64
	db.Table("scriptable_task_logs").Where("entity='server' AND entity_id=? and task like '%post_steps%' and task_status=?",
		server.ID, models.STATUS_COMPLETE).Count(&securityStepComplete)

	port := server.SshPort
	username := server.SSHUsername

	if securityStepComplete > 0 {
		port = server.NewSshPort
		username = server.NewSSHUsername
	}

	client, err := sshclient.DialWithKey(server.ServerIP+":"+strconv.Itoa(port), username, pk, pass)
	if err != nil {
		models.LogError(server.ID, "server", "Cannot SSH into server: "+server.ServerName+" - "+server.ServerIP,
			"SSH connection failed:"+err.Error(), server.TeamId)
		db.Model(&models.Server{}).Where("id", server.ID).Update("status", models.STATUS_FAILED)
		return err
	}

	defer client.Close()
	var errors int = 0

	for _, script := range scriptables {
		summary := "Ran: " + script
		var stepRanBefore int64
		db.Table("scriptable_task_logs").Where("entity='server' AND entity_id=? and task=? and task_status=?",
			server.ID, summary, models.STATUS_COMPLETE).Count(&stepRanBefore)
		if stepRanBefore > 0 {
			if utils.LogVerbose() {
				fmt.Println(" \""+summary+"\"already ran before for Server ID: .", server.ID)
			}
			continue
		}

		if utils.LogVerbose() {
			fmt.Println("Running scriptable: ", script, " for server: ", server.ServerIP)
		}

		file, err := os.Open(script)
		if err != nil {
			errors += 1
			models.LogError(server.ID, "server", err.Error()+" "+err.Error(), "Failed to run: "+script, server.TeamId)
			break
		}

		defer file.Close()

		cmd, rerr := io.ReadAll(file)
		if rerr != nil && utils.LogVerbose() {
			fmt.Println("Running scriptable: ", script, " for server: ", server.ServerIP)
		}

		if rerr != nil {
			errors += 1
			models.LogError(server.ID, "server", rerr.Error(), "Failed to run: "+script, server.TeamId)
			break
		}

		commandToRun := string(cmd)
		commandToRun = server.SubScriptableVars(commandToRun)
		commandToRun, parseError := parsers.ParseScriptImport(server, commandToRun)
		if parseError {
			if strings.Contains(commandToRun, "# exit-on-failure=yes") {
				break
			}
		}

		// For scriptables within imports
		commandToRun, parseError = parsers.ParseScriptImport(server, commandToRun)
		commandToRun = server.SubScriptableVars(commandToRun)
		if parseError {
			if strings.Contains(commandToRun, "# exit-on-failure=yes") {
				break
			}
		}

		e, output := models.RunScriptable("server", server.ID, client, commandToRun, summary, true, server.TeamId)

		if e != nil {
			errors += 1
			models.LogError(server.ID, "server",
				e.Error()+". Command output: "+output, "Failed to run: "+script, server.TeamId)
			if strings.Contains(commandToRun, "# exit-on-failure=yes") {
				break
			}
		} else {
			models.LogInfo(server.ID, "server", output, "Successfully built: "+script, server.TeamId)
		}

	}

	if errors == 0 && server.AptPackages != "" {

		packages := strings.Split(server.AptPackages, ",")
		aptCmd := "sudo apt-get install -y " + strings.Join(packages, " ")
		e, output := models.RunScriptable("server", server.ID, client, aptCmd, "Install extra apt packages.",
			true, server.TeamId)
		models.RunScriptable("server", server.ID,
			client, aptCmd, "Install extra apt packages.", true, server.TeamId)
		if e != nil {
			errors += 1
			models.LogError(server.ID, "server", e.Error(),
				"Failed to install extra apt packages.", server.TeamId)

		} else {
			models.LogInfo(server.ID, "server", output,
				"Successfully installed extra packages.", server.TeamId)
		}
	}

	if errors > 0 {
		db.Model(&models.Server{}).Where("id", server.ID).Update("status", models.STATUS_FAILED)
	} else {
		db.Model(&models.Server{}).Where("id", server.ID).Update("status", models.STATUS_COMPLETE)
	}

	return nil
}

func BuildServers(db *gorm.DB) {
	servers := models.GetQueuedBuids(5)
	var wg sync.WaitGroup
	queued := 0

	for _, s := range servers {
		if queued%5 == 0 {
			time.Sleep(20 * time.Second)
		}

		var scripts []string
		scriptable := s.ServerType
		scripts_found := utils.GetScriptables(scriptable)

		if len(scripts_found) == 0 {
			models.LogError(s.ID, "server",
				"No scriptables found for this server type.", "No scripts to run.", s.TeamId)
			continue
		}

		scripts = append(scripts, scripts_found...)

		if s.ScriptableName != "" {
			scriptables := strings.Split(s.ScriptableName, ",")
			for _, scriptable := range scriptables {
				scripts_found := utils.GetScriptables(scriptable)
				if len(scripts_found) == 0 {
					models.LogError(s.ID,
						"server",
						"Missing scriptable: "+scriptable+", please check that you've placed a folder with this name inside the scriptables folder.",
						"Missing scriptable: "+scriptable, s.TeamId)
					continue
				}

				scripts = append(scripts, scripts_found...)
			}
		}

		if len(scripts) == 0 {
			models.LogError(s.ID,
				"server",
				"Sorry, nothing to do - no scripts found to run for this server.",
				"No runnable scriptables found.", s.TeamId)
			continue
		}

		if s.Certbot == 1 {
			scripts_found := utils.GetScriptables("certbot")

			if len(scripts_found) == 0 {
				models.LogError(s.ID, "server",
					"No certbot scriptables found for this server type.", "No certbot scripts to run.", s.TeamId)
				continue
			}

			scripts = append(scripts, scripts_found...)

		}

		if s.Redis == 1 {
			scripts_found := utils.GetScriptables("redis")

			if len(scripts_found) == 0 {
				models.LogError(s.ID, "server",
					"No redis scriptables found for this server type.", "No redis scripts to run.", s.TeamId)
				continue
			}

			scripts = append(scripts, scripts_found...)
		}

		if s.Memcache == 1 {
			scripts_found := utils.GetScriptables("memcache")

			if len(scripts_found) == 0 {
				models.LogError(s.ID, "server", "No memcache scriptables found for this server type.",
					"No memcache scripts to run.", s.TeamId)
				continue
			}

			scripts = append(scripts, scripts_found...)
		}

		wg.Add(1)
		queued += 1
		runServerBuild(db, &s, scripts, &wg)
	}

	wg.Wait()
}
