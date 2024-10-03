// The firewall interface works with UFW, we run the native ufw status command and then parse the output into
// txt that we can manipulate via Scriptables GUI.
package models

import (
	"errors"
	"fmt"
	"strings"

	"plexcorp.tech/scriptable/sshclient"
)

func GetRules(client *sshclient.Client) ([]string, error) {
	firewall_rules := []string{}
	prefix := "------ firewall rules ---"
	result, err := client.Script("echo \"" + prefix + "\" && sudo ufw status numbered").SmartOutput()
	output := string(result)

	if strings.Contains(output, prefix) {
		lines := strings.Split(output, prefix)
		rules := strings.Split(lines[1], "\n")
		for _, rule := range rules {
			if strings.Contains(rule, "[") {
				var parts []string
				ruleType := ""
				if strings.Contains(rule, "ALLOW IN") {
					parts = strings.Split(rule, "ALLOW IN")
					ruleType = "ALLOW IN"
				} else if strings.Contains(rule, "ALLOW OUT") {
					parts = strings.Split(rule, "ALLOW OUT")
					ruleType = "ALLOW OUT"
				} else if strings.Contains(rule, "DENY OUT") {
					parts = strings.Split(rule, "DENY OUT")
					ruleType = "DENY OUT"
				} else if strings.Contains(rule, "DENY IN") {
					parts = strings.Split(rule, "DENY IN")
					ruleType = "DENY IN"
				}

				for i, p := range parts {

					parts[i] = strings.TrimSpace(p)
				}

				if len(parts) >= 2 {
					parts[0] = strings.ReplaceAll(parts[0], "]", "] FROM : ")
					rule = parts[0] + "   TO : " + parts[1] + "  << " + ruleType + " >>"
				}

				firewall_rules = append(firewall_rules, rule)
			}
		}

	} else {
		err = errors.New("failed to access firewall rules - please try again.")
	}

	return firewall_rules, err
}

func DeleteFirewallRule(server *ServerWithSShKey, ruleNumber int64, rule string) error {
	client, err := GetSSHClient(server, false)
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf(" echo \"y\" | sudo ufw delete %d", ruleNumber)
	out, err := client.Script(cmd).SmartOutput()
	if err == nil {
		LogInfo(server.ID, "server", string(out), fmt.Sprintf(
			"Deleted firewall rule number: %d, rule: %s", ruleNumber, rule), server.TeamId)
	}

	return err
}

func AddFirewallRule(server *ServerWithSShKey, rule string) error {
	client, err := GetSSHClient(server, false)
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf(`sudo ufw %s`, rule)
	fmt.Println(cmd)
	out, err := client.Script(cmd).SmartOutput()
	if err == nil {
		LogInfo(server.ID, "server", string(out), fmt.Sprintf("Added firewall rule: %s", rule), server.TeamId)
	} else {
		LogError(server.ID, "server", string(out), fmt.Sprintf("Failed adding firewall rule: %s", rule), server.TeamId)
	}

	return err
}
