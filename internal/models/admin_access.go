package models

// GroupAdminRoute grants a group access to an admin path prefix.
type GroupAdminRoute struct {
	GroupID  uint   `gorm:"primaryKey"`
	Path     string `gorm:"primaryKey;size:128;not null"`
	CanRead  bool   `gorm:"not null;default:true"`
	CanWrite bool   `gorm:"not null;default:false"`
}
