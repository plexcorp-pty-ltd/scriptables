// Whenever a GIT hook is fired, i.e. a push or merge. The API request will store that request in a queue table "site_queues".
// This job polls site queues and triggers a build accordingly.
package console

import (
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
	"plexcorp.tech/scriptable/models"
	"plexcorp.tech/scriptable/utils"
)

func DeployBranch(db *gorm.DB) {
	siteIDs := models.GetSitesToDeploy()
	queued := 0
	var wg sync.WaitGroup

	for _, sid := range siteIDs {
		db.Exec("UPDATE site_queues SET status = ? , updated_at = NOW() WHERE site_id = ?",
			models.STATUS_COMPLETE, sid)
		if queued%5 == 0 {
			time.Sleep(20 * time.Second)
		}

		site := models.GetSiteByIdNoTeam(sid)
		server := models.GetServer(site.ServerID, site.TeamId)
		scripts := utils.GetScriptables(site.DeployScriptables)

		if len(scripts) == 0 {
			if utils.LogVerbose() {
				fmt.Println("No scriptables to run for site:", site.SiteName)
			}
			continue
		}

		wg.Add(1)
		queued += 1
		go RunSiteBuild(db, site, &server, scripts, true, &wg)

	}

	wg.Wait()
}
