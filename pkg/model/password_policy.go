package model

import (
	"fmt"
	"time"
	"unicode"

	"golang.org/x/crypto/bcrypt"
	"k8s.io/klog/v2"
)

// PasswordHistory stores previous password hashes so a policy can reject
// reused passwords. Only the most recent PasswordHistoryCount entries are
// meaningful; older ones are trimmed on write.
type PasswordHistory struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	UserID    uint      `json:"userId" gorm:"not null;index"`
	Hash      string    `json:"-" gorm:"type:varchar(255);not null"`
	CreatedAt time.Time `json:"createdAt" gorm:"index"`
}

func recordPasswordHistory(userID uint, hash string) error {
	if DB == nil {
		return nil
	}
	if err := DB.Create(&PasswordHistory{UserID: userID, Hash: hash}).Error; err != nil {
		return err
	}
	var count int64
	if err := DB.Model(&PasswordHistory{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return err
	}
	limit := passwordHistoryLimit()
	if limit <= 0 || count <= int64(limit) {
		return nil
	}
	var oldest PasswordHistory
	trimErr := DB.Where("user_id = ?", userID).Order("created_at DESC").Offset(limit).First(&oldest).Error
	if trimErr != nil {
		klog.Warningf("Failed to trim password history for user %d: %v", userID, trimErr)
		return nil
	}
	return DB.Where("user_id = ? AND created_at <= ?", userID, oldest.CreatedAt).Delete(&PasswordHistory{}).Error
}

func passwordHistoryLimit() int {
	setting, err := GetGeneralSetting()
	if err != nil || setting == nil {
		return 0
	}
	return setting.PasswordHistoryCount
}

// ValidatePasswordPolicy enforces the configured password rules. userID > 0
// additionally enables the reuse check against recent password hashes.
func ValidatePasswordPolicy(password string, userID uint) error {
	// Deliberate fail-open: password policy is governance sugar and must
	// never block authentication flows because settings cannot be read.
	setting, _ := GetGeneralSetting()
	if setting == nil {
		return nil
	}

	minLength := setting.PasswordMinLength
	if minLength <= 0 {
		minLength = 8
	}
	if len(password) < minLength {
		return fmt.Errorf("password must be at least %d characters long", minLength)
	}
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}
	if setting.PasswordRequireUppercase && !hasUpper {
		return fmt.Errorf("password must contain an uppercase letter")
	}
	if setting.PasswordRequireLowercase && !hasLower {
		return fmt.Errorf("password must contain a lowercase letter")
	}
	if setting.PasswordRequireDigit && !hasDigit {
		return fmt.Errorf("password must contain a digit")
	}
	if setting.PasswordRequireSpecial && !hasSpecial {
		return fmt.Errorf("password must contain a special character")
	}

	if userID > 0 && setting.PasswordHistoryCount > 0 {
		var hashes []string
		historyErr := DB.Model(&PasswordHistory{}).Where("user_id = ?", userID).
			Order("created_at DESC").Limit(setting.PasswordHistoryCount).
			Pluck("hash", &hashes).Error
		if historyErr != nil {
			// Fail open: the reuse check is advisory.
			klog.Warningf("Failed to load password history for user %d: %v", userID, historyErr)
			hashes = nil
		}
		for _, hash := range hashes {
			if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil {
				return fmt.Errorf("password was used recently; choose a different one")
			}
		}
	}
	return nil
}

// IsPasswordExpired reports whether the user's password age exceeds the
// configured expiry, and when it expired.
func IsPasswordExpired(user *User) (expired bool, expiresAt *time.Time) {
	if user == nil || user.Provider != AuthProviderPassword || user.PasswordChangedAt == nil {
		return false, nil
	}
	setting, err := GetGeneralSetting()
	if err != nil || setting == nil || setting.PasswordExpireDays <= 0 {
		return false, nil
	}
	expires := user.PasswordChangedAt.Add(time.Duration(setting.PasswordExpireDays) * 24 * time.Hour)
	return time.Now().After(expires), &expires
}
