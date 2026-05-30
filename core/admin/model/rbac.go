package model

import "time"

// RBACPermission is a granular access grant referenced by roles.
type RBACPermission struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Code        string    `gorm:"size:128;uniqueIndex" json:"code"`
	Name        string    `gorm:"size:128" json:"name"`
	Group       string    `gorm:"size:64;index" json:"group"`
	Description string    `gorm:"size:512" json:"description,omitempty"`
	Builtin     bool      `json:"builtin"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RBACRole groups permissions for assignment to users.
type RBACRole struct {
	ID          uint             `gorm:"primaryKey" json:"id"`
	Code        string           `gorm:"size:64;uniqueIndex" json:"code"`
	Name        string           `gorm:"size:128" json:"name"`
	Description string           `gorm:"size:512" json:"description,omitempty"`
	Builtin     bool             `json:"builtin"`
	Permissions []RBACPermission `gorm:"many2many:rbac_role_permissions;" json:"permissions,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// RBACUser is an admin console operator account.
type RBACUser struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	Username     string     `gorm:"size:64;uniqueIndex" json:"username"`
	DisplayName  string     `gorm:"size:128" json:"display_name"`
	Email        string     `gorm:"size:255" json:"email,omitempty"`
	PasswordHash string     `gorm:"size:255" json:"-"`
	Enabled      bool       `json:"enabled"`
	Builtin      bool       `json:"builtin"`
	Roles        []RBACRole `gorm:"many2many:rbac_user_roles;" json:"roles,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
