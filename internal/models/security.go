package models

// Permission stores a registered capability in the database.
type Permission struct {
	PermissionID uint   `gorm:"primaryKey"`
	Key          string `gorm:"size:256;uniqueIndex;not null"`
	DisplayName  string `gorm:"size:256;not null"`
	Description  string `gorm:"type:text"`
	Category     string `gorm:"size:128;index"`
	Dangerous    bool   `gorm:"not null;default:false"`
	Deprecated   bool   `gorm:"not null;default:false"`
}

// RolePermission links a role to a permission key.
type RolePermission struct {
	RoleID        uint   `gorm:"primaryKey"`
	PermissionKey string `gorm:"primaryKey;size:256;not null"`
	Denied        bool   `gorm:"not null;default:false"`
}

// RoleInheritance records that ChildRoleID inherits ParentRoleID permissions.
type RoleInheritance struct {
	ChildRoleID  uint `gorm:"primaryKey"`
	ParentRoleID uint `gorm:"primaryKey"`
}
