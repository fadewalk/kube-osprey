package model

import "time"

// UserGroup is a locally managed group used for batch authorization.
// Role assignments with SubjectType "group" and Subject == Name apply to
// every active member.
type UserGroup struct {
	Model

	Name        string `json:"name" gorm:"type:varchar(100);uniqueIndex;not null"`
	Description string `json:"description" gorm:"type:text"`
	Enabled     bool   `json:"enabled" gorm:"type:boolean;not null;default:true"`

	Members []UserGroupMember `json:"members" gorm:"foreignKey:GroupID;constraint:OnDelete:CASCADE"`
}

type UserGroupMember struct {
	Model

	GroupID uint `json:"groupId" gorm:"index;not null;constraint:OnDelete:CASCADE"`
	UserID  uint `json:"userId" gorm:"not null;uniqueIndex:idx_user_group_members"`
	// ExpiresAt bounds the membership; nil means the membership never
	// expires. Expired members lose group-derived permissions.
	ExpiresAt *time.Time `json:"expiresAt"`

	User User `json:"user" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

func GetUserGroupByName(name string) (*UserGroup, error) {
	var group UserGroup
	if err := DB.Where("name = ?", name).First(&group).Error; err != nil {
		return nil, err
	}
	return &group, nil
}
