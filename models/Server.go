package models

import (
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
	"plexcorp.tech/scriptable/utils"
)

type Server struct {
	gorm.Model
	ID                int64     `gorm:"column:id"`
	ServerName        string    `gorm:"column:server_name;type:varchar(100)"`
	ServerType        string    `gorm:"column:server_type;type:varchar(50)"`
	ServerIP          string    `gorm:"column:server_ip;type:varchar(100)"`
	PrivateServerIP   string    `gorm:"column:private_server_ip;type:varchar(100)"`
	SSHKeyId          int64     `gorm:"column:ssh_key_id"`
	SSHUsername       string    `gorm:"column:ssh_username;type:varchar(100)"`
	NewSSHUsername    string    `gorm:"column:new_ssh_username;type:varchar(100)"`
	SshPort           int       `gorm:"column:ssh_port"`
	NewSshPort        int       `gorm:"column:new_ssh_port"`
	Redis             int       `gorm:"column:redis;type:tinyint(3)"`
	Certbot           int       `gorm:"column:certbot;type:tinyint(3)"`
	Memcache          int       `gorm:"column:memcache;type:tinyint(3)"`
	MySql             int       `gorm:"column:mysql;type:tinyint(3)"`
	MySqlRootPassword string    `gorm:"column:mysql_root_password;type:varchar(100)"`
	PhpVersion        string    `gorm:"column:php_version;type:varchar(100)"`
	WebserverType     string    `gorm:"column:webserver_type;type:varchar(100)"`
	ScriptableName    string    `gorm:"column:scriptable_name;type:varchar(50)"`
	Status            string    `gorm:"column:status;type:varchar(100)"`
	AptPackages       string    `gorm:"column:apt_packages"`
	CreatedAt         time.Time `gorm:"column:created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at"`
	TeamId            int64     `gorm:"column:team_id"`
}

type ServerType struct {
	Slug        string
	Name        string
	Description string
}

type ServerWithSShKey struct {
	ID                int64
	ServerName        string
	ServerType        string
	ServerIP          string
	PrivateServerIP   string
	SshPort           int
	NewSshPort        int
	SSHUsername       string
	NewSSHUsername    string
	Redis             int
	Certbot           int
	Memcache          int
	MySql             int
	MySqlRootPassword string
	PhpVersion        string
	WebserverType     string
	ScriptableName    string
	Status            string
	PrivateKey        string
	PublicKey         string
	Passphrase        string
	AptPackages       string
	TeamId            int64
}

func GetServerTypes() []ServerType {

	return []ServerType{
		{Slug: "nginx", Name: "Nginx web server", Description: "Standalone NGINX webserver"},
		//{Slug: "apache2", Name: "Apache2 web server", Description: "Standalone NGINX webserver"},
		{Slug: "mysql", Name: "MySQL database server", Description: "Standalone MySQL server"},
		{Slug: "lemp", Name: "LEMP application server", Description: "All the key components to run a standard PHP application with nginx as the web server."},
		//{Slug: "lamp", Name: "LAMP application server", Description: "All the key components to run a standard PHP application with apache2 as the web server."},
		//{Slug: "php", Name: "PHP application server", Description: "PHP server with php-fpm and other standard php libraries."},
		{Slug: "cache", Name: "Standalone cache server", Description: "Redis or memcache to store sessions and cache data."},
		{Slug: "scriptable", Name: "Scriptable server", Description: "Deploy scriptable to a cloud server."},
	}

}

func GetServersList(page int, perPage int, search string, status string, teamId int64) []Server {
	offset := (page - 1) * perPage
	var servers []Server

	query := GetDB().Where("team_id=?", teamId).Limit(perPage).Offset(offset)

	if search != "" {
		searchQuery := search + "%"
		query = query.Where("(server_name LIKE ? OR server_ip LIKE ?) ", searchQuery, searchQuery)
	}

	if status != "" && status != "all" {
		query = query.Where("status = ? and team_id=?", status, teamId)
	}

	query.Find(&servers)

	return servers
}

func ValidateForm(gctx echo.Context, s *Server) []string {
	errors := []string{}

	s.ServerName = gctx.FormValue("server_name")
	s.ServerIP = gctx.FormValue("server_ip")
	s.PrivateServerIP = gctx.FormValue("private_server_ip")
	s.PhpVersion = gctx.FormValue("php_version")
	s.SSHKeyId, _ = strconv.ParseInt(gctx.FormValue("ssh_key_id"), 10, 64)

	s.Certbot = 0
	s.Redis = 0
	s.Memcache = 0

	if gctx.FormValue("certbot") == "on" {
		s.Certbot = 1
	}

	if gctx.FormValue("redis") == "on" {
		s.Redis = 1
	}

	if gctx.FormValue("memcache") == "on" {
		s.Memcache = 1
	}

	if s.SSHKeyId == 0 {
		errors = append(errors, "Please select an SSH key.")
	}

	s.MySql = 0

	if s.ServerType == "lemp" || s.ServerType == "lamp" || s.ServerType == "mysql" {
		s.MySql = 1
		s.MySqlRootPassword = gctx.FormValue("mysql_root_password")
		MysqlConfirmRootPassword := gctx.FormValue("confirm_mysql_root_password")

		if len(s.MySqlRootPassword) < 6 {
			errors = append(errors, "MYSQL root password must be at least 6 characters.")
		}

		if s.MySqlRootPassword != MysqlConfirmRootPassword {
			errors = append(errors, "MYSQL root password and confirm root password do not match.")
		}
	}

	if s.ServerType == "cache" && s.Redis == 0 && s.Memcache == 0 {
		errors = append(errors, "Please select a cache server type either redis or memcache.")
	}

	s.SSHUsername = gctx.FormValue("ssh_username")
	s.NewSSHUsername = gctx.FormValue("new_ssh_username")
	s.ScriptableName = gctx.FormValue("scriptable_name")
	s.SshPort, _ = strconv.Atoi(gctx.FormValue("ssh_port"))
	s.NewSshPort, _ = strconv.Atoi(gctx.FormValue("new_ssh_port"))
	s.AptPackages = gctx.FormValue("apt_packages")

	if s.SSHUsername == "" || len(s.SSHUsername) < 3 {
		errors = append(errors, "Please input your server SSH username, usually this is root. We'll need this intially to connect to this server.")
	}

	if strings.Count(s.ServerIP, ".") < 3 {
		errors = append(errors, "IP Address seems invalid. Please enter a valid IP e.g. : 192.168.10.10")
	}

	if len(s.NewSSHUsername) < 3 {
		errors = append(errors, "Please specify a new SSH username e.g. \"developer\". This allows us to isolate SSH connections to this user instead of root or the primary user.")
	}

	if (s.ServerType == "lemp" || s.ServerType == "lamp") && s.PhpVersion == "" {

		errors = append(errors, "Please specify a default PHP version to install.")
	}

	if s.ServerType == "lemp" || s.ServerType == "nginx" {
		s.WebserverType = "nginx"
	} else if s.ServerType == "lamp" || s.ServerType == "apache2" {
		s.WebserverType = "apache2"
	}

	return errors
}

func GetQueuedBuids(limit int) []ServerWithSShKey {
	if limit == 0 {
		limit = 10
	}

	rows, _ := GetDB().Raw(
		`SELECT s.ID, s.server_name, s.server_type, s.server_ip,s.private_server_ip,
		s.ssh_username,s.new_ssh_username,s.redis,s.certbot,s.memcache,
		s.mysql,s.mysql_root_password,s.php_version,s.webserver_type, s.scriptable_name,s.status,s.ssh_port,s.new_ssh_port,
		s.apt_packages,k.private_key, k.public_key, k.passphrase, s.team_id
		FROM servers s 
		JOIN ssh_keys k ON(k.ID = s.ssh_key_id)
		WHERE s.status = ?
		ORDER BY s.created_at ASC
		LIMIT ?
		`, STATUS_QUEUED, limit).Rows()

	defer rows.Close()

	var servers []ServerWithSShKey
	for rows.Next() {

		var server ServerWithSShKey
		rows.Scan(
			&server.ID,
			&server.ServerName,
			&server.ServerType,
			&server.ServerIP,
			&server.PrivateServerIP,
			&server.SSHUsername,
			&server.NewSSHUsername,
			&server.Redis,
			&server.Certbot,
			&server.Memcache,
			&server.MySql,
			&server.MySqlRootPassword,
			&server.PhpVersion,
			&server.WebserverType,
			&server.ScriptableName,
			&server.Status,
			&server.SshPort,
			&server.NewSshPort,
			&server.AptPackages,
			&server.PrivateKey,
			&server.PublicKey,
			&server.Passphrase,
			&server.TeamId,
		)

		servers = append(servers, server)
	}

	return servers
}

func GetServer(serverId int64, teamId int64) ServerWithSShKey {
	var server ServerWithSShKey

	row := GetDB().Raw(
		`SELECT s.ID, s.server_name, s.server_type, s.server_ip,s.private_server_ip,
		s.ssh_username,s.new_ssh_username,s.redis,s.certbot,s.memcache,
		s.mysql,s.mysql_root_password,s.php_version,s.webserver_type, s.scriptable_name,s.status,s.ssh_port,s.new_ssh_port,
		s.apt_packages,k.private_key, k.public_key, k.passphrase, s.team_id
		FROM servers s 
		JOIN ssh_keys k ON(k.ID = s.ssh_key_id)
		WHERE s.ID = ?
		`, serverId).Row()

	row.Scan(
		&server.ID,
		&server.ServerName,
		&server.ServerType,
		&server.ServerIP,
		&server.PrivateServerIP,
		&server.SSHUsername,
		&server.NewSSHUsername,
		&server.Redis,
		&server.Certbot,
		&server.Memcache,
		&server.MySql,
		&server.MySqlRootPassword,
		&server.PhpVersion,
		&server.WebserverType,
		&server.ScriptableName,
		&server.Status,
		&server.SshPort,
		&server.NewSshPort,
		&server.AptPackages,
		&server.PrivateKey,
		&server.PublicKey,
		&server.Passphrase,
		&server.TeamId,
	)

	return server
}

func GetServerSimple(serverID int64, teamId int64) *Server {
	var server *Server
	GetDB().Where("id=? and team_id=?", serverID, teamId).First(&server)
	return server
}

func GetServerByIp(serverIP string, teamId int64) ServerWithSShKey {
	var server ServerWithSShKey

	row := GetDB().Raw(
		`SELECT s.ID, s.server_name, s.server_type, s.server_ip,s.private_server_ip,
		s.ssh_username,s.new_ssh_username,s.redis,s.certbot,s.memcache,
		s.mysql,s.mysql_root_password,s.php_version,s.webserver_type, s.scriptable_name,s.status,s.ssh_port,s.new_ssh_port,
		s.apt_packages,k.private_key, k.public_key, k.passphrase,s.team_id
		FROM servers s
		JOIN ssh_keys k ON(k.ID = s.ssh_key_id)
		WHERE s.server_ip = ?
		`, serverIP).Row()

	row.Scan(
		&server.ID,
		&server.ServerName,
		&server.ServerType,
		&server.ServerIP,
		&server.PrivateServerIP,
		&server.SSHUsername,
		&server.NewSSHUsername,
		&server.Redis,
		&server.Certbot,
		&server.Memcache,
		&server.MySql,
		&server.MySqlRootPassword,
		&server.PhpVersion,
		&server.WebserverType,
		&server.ScriptableName,
		&server.Status,
		&server.SshPort,
		&server.NewSshPort,
		&server.AptPackages,
		&server.PrivateKey,
		&server.PublicKey,
		&server.Passphrase,
		&server.TeamId,
	)

	return server
}

func (s *ServerWithSShKey) SubScriptableVars(script string) string {
	script = strings.ReplaceAll(script, "#username#", s.NewSSHUsername)
	script = strings.ReplaceAll(script, "#MYSQL_ROOT_PASSWORD#", utils.Decrypt(s.MySqlRootPassword))
	script = strings.ReplaceAll(script, "#SSH_PORT#", strconv.Itoa(s.SshPort))
	script = strings.ReplaceAll(script, "#NEW_SSH_PORT#", strconv.Itoa(s.NewSshPort))
	script = strings.ReplaceAll(script, "#PHP_VERSION#", s.PhpVersion)
	script = strings.ReplaceAll(script, "#PUBKEY#", utils.Decrypt(s.PublicKey))
	script = strings.ReplaceAll(script, "#SERVER_IP#", s.ServerIP)

	FPMPort := "90" + strings.ReplaceAll(s.PhpVersion, ".", "")
	script = strings.ReplaceAll(script, "#FPM_PORT#", FPMPort)
	return script
}

func IsMyServer(serverID int64, teamID int64) bool {
	totalFound := int64(0)

	GetDB().Table("servers").Where("id=? and team_id=?", serverID, teamID).Count(&totalFound)

	return totalFound > 0
}
