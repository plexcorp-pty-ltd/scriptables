// We extend bash slighly to allow for importing code across scriptables. Anything placed in ./scriptables/__shared - can then
// be imported anywhere else.
package parsers

import (
	"io"
	"os"
	"strconv"
	"strings"

	"plexcorp.tech/scriptable/models"
)

func parseServerImport(lineNumber int, script string, server *models.ServerWithSShKey, line string) (string, bool) {
	originalLine := line
	line = strings.ReplaceAll(line, "SCRIPTABLE::IMPORT", "")
	line = strings.ReplaceAll(line, " ", "")
	imported, err := os.Open("./scriptables/__shared/" + line + ".sh")
	if err != nil {
		models.LogError(server.ID, "server", "Failed to import scriptable code at line: "+strconv.Itoa(lineNumber)+". "+err.Error(),
			"Failed to parse scriptable: "+script, server.TeamId)
		return script, true
	}

	defer imported.Close()

	importedCmd, rerr := io.ReadAll(imported)

	if rerr != nil {
		models.LogError(server.ID, "server", "Failed to import scriptable code at line: "+strconv.Itoa(lineNumber)+". "+err.Error(),
			"Failed to parse scriptable: "+script, server.TeamId)
		return script, true
	}

	script = strings.ReplaceAll(script, originalLine, string(importedCmd))

	return script, false
}

func ParseScriptImport(server *models.ServerWithSShKey, script string) (string, bool) {
	var failed bool = false
	if strings.Contains(script, "SCRIPTABLE::") {
		lines := strings.Split(script, "\n")
		for i, line := range lines {
			if strings.Contains(line, "SCRIPTABLE::IMPORT") {
				script, failed = parseServerImport(i, script, server, line)
				if failed {
					return script, failed
				}
			}
		}
	}

	return script, failed
}

func ParseSiteScriptable(site *models.Site, script string) (string, bool) {
	var failed bool = false

	if strings.Contains(script, "SCRIPTABLE::") {
		lines := strings.Split(script, "\n")
		for lineNumber, line := range lines {
			if strings.Contains(line, "SCRIPTABLE::IMPORT") {
				originalLine := line
				line = strings.ReplaceAll(line, "SCRIPTABLE::IMPORT", "")
				line = strings.ReplaceAll(line, " ", "")
				imported, err := os.Open("./scriptables/__shared/" + line + ".sh")
				if err != nil {
					models.LogError(site.ID, "server", "Failed to import scriptable code at line: "+strconv.Itoa(lineNumber)+". "+err.Error(),
						"Failed to parse scriptable: "+script, site.TeamId)
					return script, true
				}

				defer imported.Close()

				importedCmd, rerr := io.ReadAll(imported)

				if rerr != nil {
					models.LogError(site.ID, "site", "Failed to import scriptable code at line: "+strconv.Itoa(lineNumber)+". "+err.Error(),
						"Failed to parse scriptable: "+script, site.TeamId)
					return script, true
				}

				script = strings.ReplaceAll(script, originalLine, string(importedCmd))

			}
		}

	}

	return script, failed
}
