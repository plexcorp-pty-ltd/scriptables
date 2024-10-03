package models

import (
	"time"

	"github.com/robfig/cron"
	"gorm.io/gorm"
)

type Cron struct {
	gorm.Model
	ID             int64     `gorm:"column:id"`
	User           string    `gorm:"column:user"`
	Task           string    `gorm:"column:task;type:varchar(255)"`
	CronExpression string    `gorm:"column:cron_expression;type:varchar(100)"`
	CronName       string    `gorm:"column:cron_name"`
	ServerID       int64     `gorm:"column:server_id"`
	Status         string    `gorm:"column:status"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
	TeamID         int64     `gorm:"column:team_id"`
}

type CronServer struct {
	Cron
	ServerName string
}

func IsValidCronExpression(input string) bool {
	_, err := cron.ParseStandard(input)
	return err == nil
}

func GetCrons(page, perPage int, search string, teamId int64) []CronServer {
	offset := (page - 1) * perPage
	var crons []CronServer

	query := GetDB().Table("crons").Select(
		"crons.*,servers.server_name",
	).Where("crons.team_id = ?", teamId).Joins(" JOIN servers ON (crons.server_id = servers.ID)")

	if search != "" {
		searchQuery := search + "%"
		query = query.Where("crons.name LIKE ?", searchQuery)
	}
	query.Limit(perPage).Offset(offset).Find(&crons)
	return crons
}
