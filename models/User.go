package models

import (
	"fmt"
	"time"

	"github.com/noirbizarre/gonja"
	"gorm.io/gorm"
	"plexcorp.tech/scriptable/utils"
)

type User struct {
	gorm.Model
	Name          string    `gorm:"column:name;type:varchar(100)"`
	ID            int64     `gorm:"column:id"`
	Email         string    `gorm:"column:email;type:varchar(255)"`
	Password      string    `gorm:"column:password;type:varchar(155)"`
	Verified      int       `gorm:"column:verified;type:tinyint(3)"`
	ResetToken    string    `gorm:"column:reset_token;type:varchar(155)"`
	TwoFactor     int       `gorm:"column:two_factor;type:tinyint(3)"`
	TwoFactorCode string    `gorm:"column:two_factor_code;type:varchar(155)"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
	TeamId        int64     `gorm:"column:team_id"`
}

type FailedLogins struct {
	gorm.Model
	DateTime          time.Time `gorm:"column:date_time"`
	IPAddress         string    `gorm:"column:ip_address;type:varchar(100)"`
	AttemptedUserName string    `gorm:"column:attempted_user_name;type:varchar(100)"`
}

func Authenticate(email string, password string, IPAddress string) (User, error) {

	var totalTries int64
	fiveMinutesAgo := time.Now().Add(-5 * time.Minute)
	GetDB().Table("failed_logins").
		Where("ip_address = ? AND created_at >= ?", IPAddress, fiveMinutesAgo).
		Count(&totalTries)

	if totalTries >= 5 {
		return User{}, fmt.Errorf("Sorry, too many failed login attempts. Please wait 5 minutes and try again.")
	}

	var user User
	GetDB().Where("email =? AND verified = 1", email).First(&user)

	if user.ID != 0 && utils.CheckPasswordHash(password, user.Password) {
		return user, nil
	}

	failedAttempt := FailedLogins{
		DateTime:          time.Now(),
		IPAddress:         IPAddress,
		AttemptedUserName: email,
	}

	GetDB().Create(&failedAttempt)

	return User{}, fmt.Errorf("Sorry, login failed - please try again.")
}

func IsValidEmail(email string, IPAddress string) bool {
	var user User
	GetDB().Where("email =?", email).First(&user)
	return user.Email == email
}

func SendPasswordResetToken(email string, subject string, template string) {

	token := utils.GenToken()
	GetDB().Table("users").Where("email =?", email).Update("reset_token", token)

	var user User
	GetDB().Table("users").Where("email =?", email).First(&user)
	vars := gonja.Context{
		"subject": subject,
		"name":    user.Name,
		"token":   token,
		"email":   user.Email,
	}

	utils.SendEmail(subject, "", []string{user.Email}, vars, template)
}

func GetUserById(id int64) User {
	var user User
	GetDB().Where("id=?", id).First(&user)
	return user
}

func GetUserByEmailToken(email string, token string) User {
	var user User
	GetDB().Where("email =? AND reset_token != '' AND reset_token = ?", email, token).First(&user)
	return user
}

func UpdateUserPassword(user *User) {
	GetDB().Save(user)
}

func GetUsersList(page int, perPage int, search string, teamId int64) []User {
	offset := (page - 1) * perPage
	var users []User

	if search != "" {
		GetDB().Limit(perPage).Offset(offset).Where(
			"(email LIKE ? OR name LIKE ?) AND team_id=?",
			"%"+search+"%",
			"%"+search+"%",
			teamId,
		).Find(&users)

	} else {
		GetDB().Limit(perPage).Where("team_id=?", teamId).Offset(offset).Find(&users)
	}

	return users
}

func ToggleUserStatus(userId int64, userStatus int, teamId int64) error {
	GetDB().Table("users").Where("id=? and team_id=?", userId, teamId).Update("verified", userStatus)
	var updated int64
	GetDB().Table("users").Where("id=? and verified=? and team_id=?", userId, userStatus, teamId).Count(&updated)
	if updated == 1 {
		return nil
	}

	return fmt.Errorf("Sorry, failed to update user. Please try again.")
}
