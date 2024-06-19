package models

import (
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"plexcorp.tech/scriptable/utils"
)

type SiteQueue struct {
	ID int64 `gorm:"column:id"`

	SiteID    int64
	Status    string
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

type Site struct {
	gorm.Model
	ID                     int64     `gorm:"column:id"`
	Domain                 string    `gorm:"column:domain;type:varchar(255)"`
	ScriptableName         string    `gorm:"column:scriptable_name;type:varchar(100)"`
	DeployScriptables      string    `gorm:"column:deploy_scriptables;type:varchar(100)"`
	SiteName               string    `gorm:"column:site_name"`
	ServerID               int64     `gorm:"column:server_id"`
	SSHKeyId               int64     `gorm:"column:ssh_key_id"`
	Webroot                string    `gorm:"column:webroot;type:varchar(100)"`
	PhpVersion             string    `gorm:"column:php_version;type:varchar(50)"`
	LetsEncryptCertificate int       `gorm:"column:lets_encrypt_certificate;type:tinyint(3)"`
	MysqlPassword          string    `gorm:"column:mysql_password;type:varchar(155)"`
	GitURL                 string    `gorm:"column:git_url;type:varchar(255)"`
	Branch                 string    `gorm:"column:branch;type:varchar(100)"`
	Environment            string    `gorm:"column:environment;type:varchar(50)"`
	Status                 string    `gorm:"column:status;type:varchar(50)"`
	DeployToken            string    `gorm:"column:deploy_token;type:varchar(255)"`
	CreatedAt              time.Time `gorm:"column:created_at"`
	UpdatedAt              time.Time `gorm:"column:updated_at"`
	TeamId                 int64     `gorm:"column:team_id"`
}

type SiteJoinServer struct {
	ID                     int64
	SiteName               string
	Domain                 string
	ScriptableName         string
	ServerID               int64
	SSHKeyId               int64
	ServerName             string
	DeployToken            string
	Webroot                string
	PhpVersion             string
	LetsEncryptCertificate int
	MysqlPassword          string `gorm:"type:varchar(155)"`
	GitURL                 string
	Branch                 string
	Environment            string
	Status                 string
	CreatedAt              time.Time
	UpdatedAt              time.Time
	TeamId                 int64
}

func GetSitesList(page, perPage int, search, status string, teamId int64) []SiteJoinServer {
	offset := (page - 1) * perPage
	var sites []SiteJoinServer

	query := GetDB().Table("sites").Select("sites.*, servers.server_name").Where("sites.team_id=?", teamId).Joins(
		"left join servers on servers.ID = sites.server_id")

	if search != "" {
		searchQuery := search + "%"
		query = query.Where("sites.domain LIKE ?", searchQuery)
	}

	if status != "" && status != "all" {
		query = query.Where("sites.status = ?", status)
	}

	query.Limit(perPage).Offset(offset).Find(&sites)

	return sites
}

func GetSiteByTokenAndId(token string, id int64) *Site {
	var site *Site
	GetDB().Where("deploy_token=? and id=?", token, id).First(&site)
	return site
}

func GetSiteById(id int64, teamId int64) *Site {
	var site *Site
	GetDB().Where("id=? and team_id = ?", id, teamId).First(&site)
	return site
}

func GetSiteByIdNoTeam(id int64) *Site {
	var site *Site
	GetDB().Where("id=?", id).First(&site)
	return site
}

func GetSitesToProcess() []Site {
	var sites []Site

	GetDB().Table("sites").Where("status=?", STATUS_QUEUED).Scan(&sites)

	return sites
}

func GetSitesToDeploy() []int64 {
	var siteIds []int64

	GetDB().Table("site_queues").Select("site_id").Where("status=?", STATUS_QUEUED).Scan(&siteIds)

	return siteIds
}

func (site *Site) SubScriptableVars(server *ServerWithSShKey, script string) string {
	username := utils.Slugify(site.SiteName)
	script = strings.ReplaceAll(script, "#USERNAME#", server.NewSSHUsername)
	script = strings.ReplaceAll(script, "#SITE_NAME#", site.SiteName)
	script = strings.ReplaceAll(script, "#SITE_SLUG#", username)
	script = strings.ReplaceAll(script, "#MYSQL_ROOT_PASSWORD#", utils.Decrypt(server.MySqlRootPassword))
	script = strings.ReplaceAll(script, "#MYSQL_PASSWORD#", utils.Decrypt(site.MysqlPassword))
	script = strings.ReplaceAll(script, "#PHP_VERSION#", site.PhpVersion)
	script = strings.ReplaceAll(script, "#BRANCH#", site.Branch)
	script = strings.ReplaceAll(script, "#GIT_URL#", site.GitURL)
	script = strings.ReplaceAll(script, "#ENVIRONMENT#", strings.ReplaceAll(site.Environment, ".env", ""))
	script = strings.ReplaceAll(script, "#WEBROOT#", site.Webroot)
	script = strings.ReplaceAll(script, "#DOMAIN#", site.Domain)
	script = strings.ReplaceAll(script, "#KEY_PATH#", GetSiteDeployPubKeyPath(site.ID, site.SiteName, username))
	script = strings.ReplaceAll(script, "#USER_DIRECTORY#", "/home/"+username)

	var user User
	GetDB().Table("users").Where("id", 1).Scan(&user)
	script = strings.ReplaceAll(script, "#NOTIFY_EMAIL#", user.Email)

	FPMPort := "90" + strings.ReplaceAll(site.PhpVersion, ".", "")
	script = strings.ReplaceAll(script, "#FPM_PORT#", FPMPort)

	return script
}

func (site *Site) SubScriptableSiteVarsOnly(script string) string {
	username := utils.Slugify(site.SiteName)
	script = strings.ReplaceAll(script, "#SITE_NAME#", site.SiteName)
	script = strings.ReplaceAll(script, "#SITE_SLUG#", username)
	script = strings.ReplaceAll(script, "#PHP_VERSION#", site.PhpVersion)
	script = strings.ReplaceAll(script, "#BRANCH#", site.Branch)
	script = strings.ReplaceAll(script, "#GIT_URL#", site.GitURL)
	script = strings.ReplaceAll(script, "#ENVIRONMENT#", strings.ReplaceAll(site.Environment, ".env", ""))
	script = strings.ReplaceAll(script, "#WEBROOT#", site.Webroot)
	script = strings.ReplaceAll(script, "#DOMAIN#", site.Domain)
	script = strings.ReplaceAll(script, "#KEY_PATH#", GetSiteDeployPubKeyPath(site.ID, site.SiteName, username))
	script = strings.ReplaceAll(script, "#USER_DIRECTORY#", "/home/"+username)
	return script
}

func GetSiteDeployPubKeyPath(id int64, siteName string, username string) string {
	return "/home/" + username + "/.ssh/" + strconv.Itoa(int(id)) + siteName
}
