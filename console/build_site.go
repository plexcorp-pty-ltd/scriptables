// This job builds Laravel, and other projects types on servers.
package console

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
	"plexcorp.tech/scriptable/models"
	"plexcorp.tech/scriptable/parsers"
	"plexcorp.tech/scriptable/utils"
)

func getScriptables(site *models.Site) []string {
	return utils.GetScriptables(site.ScriptableName)
}

func RunSiteBuild(db *gorm.DB, site *models.Site, server *models.ServerWithSShKey, scriptables []string,
	logSteps bool, wg *sync.WaitGroup) error {
	db.Model(&models.Site{}).Where("id", site.ID).Update("status", models.STATUS_RUNNING)

	defer func() {
		if wg != nil {
			wg.Done()
		}

		if r := recover(); r != nil {
			db.Model(&models.Server{}).Where("id", server.ID).Update("status", "Failed")
			fmt.Println("Recovered from serious server build failure. Please check the logs for server:", server.ServerName, r)
		}

	}()

	client, err := models.GetSSHClient(server, false)
	if err != nil {
		models.LogError(server.ID, "server",
			"Cannot SSH into server: "+server.ServerName+" - "+server.ServerIP, "SSH connection failed: "+err.Error(), server.TeamId)
		db.Model(&models.Site{}).Where("id", site.ID).Update("status", models.STATUS_FAILED)
		return err
	}

	defer client.Close()
	var errors int = 0

	for _, script := range scriptables {
		fmt.Println("Running scriptable: ", script, " for server: ", site.SiteName, "on ", server.ServerName)

		summary := "Ran: " + script
		if logSteps {
			var stepRanBefore int64
			db.Table("scriptable_task_logs").Where("entity='server' AND entity_id=? and task=? and task_status=?", server.ID, summary, models.STATUS_COMPLETE).Count(&stepRanBefore)
			if stepRanBefore > 0 {
				if utils.LogVerbose() {
					fmt.Println(" \"" + summary + "\"already ran before.")
				}
				continue
			}
		}

		file, err := os.Open(script)
		if err != nil {
			errors += 1
			models.LogError(site.ID, "site", err.Error()+" "+err.Error(), "Failed to run: "+script, server.TeamId)
			break
		}

		defer file.Close()

		cmd, rerr := io.ReadAll(file)
		if rerr != nil {
			if utils.LogVerbose() {
				fmt.Println(rerr)
			}
			errors += 1
			models.LogError(site.ID, "site", rerr.Error(), "Failed to run: "+script, site.TeamId)
			break
		}

		commandToRun := string(cmd)
		commandToRun = site.SubScriptableVars(server, commandToRun)

		commandToRun, parseError := parsers.ParseSiteScriptable(site, commandToRun)
		if parseError {
			if strings.Contains(commandToRun, "exit-on-failure=yes") {
				break
			}
		}

		// For scriptables within imports
		commandToRun = site.SubScriptableVars(server, commandToRun)

		if parseError {
			if strings.Contains(commandToRun, "exit-on-failure=yes") {
				break
			}
		}

		e, output := models.RunScriptable("site", site.ID, client, commandToRun, summary, true, site.TeamId)

		if e != nil {
			errors += 1
			models.LogError(site.ID, "site",
				e.Error()+". Command output: "+output, "Failed to run: "+script, site.TeamId)
			if strings.Contains(commandToRun, "# exit-on-failure=yes") {
				break
			}
		} else {
			models.LogInfo(site.ID, "site", output, "Successfully ran deploy: "+script, site.TeamId)
		}

	}

	if errors > 0 {
		db.Model(&models.Site{}).Where("id", site.ID).Update("status", models.STATUS_FAILED)
	} else {
		db.Model(&models.Site{}).Where("id", site.ID).Update("status", models.STATUS_COMPLETE)
	}

	return nil
}

func BuildSites(db *gorm.DB) {
	sites := models.GetSitesToProcess()
	queued := 0
	var wg sync.WaitGroup

	for _, site := range sites {
		if queued%5 == 0 {
			time.Sleep(20 * time.Second)
		}
		server := models.GetServer(site.ServerID, site.TeamId)
		if server.Status != models.STATUS_COMPLETE {
			if utils.LogVerbose() {
				fmt.Println("Server " + server.ServerName + " is not ready to deploy sites. Please check build log for server.")
			}
			continue
		}
		scriptables := getScriptables(&site)

		if len(scriptables) == 0 {
			if utils.LogVerbose() {
				fmt.Println("No scriptables to run for site:", site.SiteName)
			}
			continue
		}

		if site.LetsEncryptCertificate == 1 {
			scriptables = append(scriptables, utils.GetScriptables("ssl_setup")[0])
		}

		wg.Add(1)
		queued += 1
		RunSiteBuild(db, &site, &server, scriptables, true, &wg)
	}

	wg.Wait()
}
