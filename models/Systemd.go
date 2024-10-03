package models

import (
	"time"

	"gorm.io/gorm"
)

type SystemdService struct {
	gorm.Model
	ID         int64     `gorm:"column:id"`
	Name       string    `gorm:"column:name"`
	Status     string    `gorm:"column:status"`
	Command    string    `gorm:"column:command"`
	Scriptable string    `gorm:"column:scriptable"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
	TeamId     int64     `gorm:"column:team_id"`
}

func GetSiteWorkers(siteId int64, teamId int64) []SystemdService {
	var services []SystemdService
	GetDB().Raw("SELECT id, name,status,command,scriptable, created_at, updated_at, team_id FROM systemd_services WHERE site_id = ? AND team_id = ?",
		siteId, teamId).Scan(&services)

	return services
}
