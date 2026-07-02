package models

import "time"

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
)

type Authenticator struct {
	AuthID        uint   `gorm:"primaryKey"`
	Name          string `gorm:"size:128;uniqueIndex;not null"`
	Status        Status `gorm:"size:16;not null;default:active"`
	Configuration string `gorm:"type:text"`
}

type User struct {
	UserID     uint   `gorm:"primaryKey"`
	GivenName  string `gorm:"size:128"`
	FamilyName string `gorm:"size:128"`
	Email      string `gorm:"size:256;uniqueIndex"`
	Username   string `gorm:"size:128;uniqueIndex;not null"`
	Locked     bool   `gorm:"not null;default:false"`
	Validated  bool   `gorm:"not null;default:false"`
	Hash       string `gorm:"size:256"`
	Status     Status `gorm:"size:16;not null;default:active"`
	AuthID     *uint
	Auth       *Authenticator
	Groups     []Group `gorm:"many2many:user_groups;"`
}

type Profile struct {
	ProfileID uint   `gorm:"primaryKey"`
	Name      string `gorm:"size:128;uniqueIndex;not null"`
	Fields    []ProfileField
}

type ProfileField struct {
	ProfileFieldID uint   `gorm:"primaryKey"`
	ProfileID      uint   `gorm:"index;not null"`
	Name           string `gorm:"size:128;not null"`
	Type           string `gorm:"size:64;not null"`
	Sort           int    `gorm:"not null;default:0"`
	Status         Status `gorm:"size:16;not null;default:active"`
	Configuration  string `gorm:"type:text"`
}

type UserProfile struct {
	UserProfileID uint `gorm:"primaryKey"`
	UserID        uint `gorm:"index;not null"`
	ProfileID     uint `gorm:"index;not null"`
}

type ProfileUserFieldValue struct {
	UserID  uint   `gorm:"primaryKey"`
	FieldID uint   `gorm:"primaryKey"`
	Value   string `gorm:"type:text"`
}

type Extension struct {
	ExtensionID uint   `gorm:"primaryKey"`
	Name        string `gorm:"size:128;uniqueIndex;not null"`
	Title       string `gorm:"size:256"`
	Description string `gorm:"type:text"`
	Version     string `gorm:"size:64"`
	MenuName    string `gorm:"size:128"`
	Socket      string `gorm:"size:512;not null"`
	Sort        int    `gorm:"not null;default:0"`
	Status      Status `gorm:"size:16;not null;default:inactive"`
	Installed   bool   `gorm:"not null;default:false"`
}

type RouteType string

const (
	RouteTypeURL                RouteType = "Url"
	RouteTypeExtension          RouteType = "Extension"
	RouteTypeExtensionEndpoint  RouteType = "Extension Endpoint"
	RouteTypeLocalFile          RouteType = "Local File"
	RouteTypeController         RouteType = "Controller"
)

type Route struct {
	RouteID       uint      `gorm:"primaryKey"`
	Name          string    `gorm:"size:128;not null"`
	Path          string    `gorm:"size:512;uniqueIndex;not null"`
	Type          RouteType `gorm:"size:32;not null"`
	Status        Status    `gorm:"size:16;not null;default:active"`
	Target        string    `gorm:"size:1024"`
	ExtensionName         string    `gorm:"size:128"`
	ExtensionPageID       string    `gorm:"size:128"`
	ExtensionEndpointID   string    `gorm:"size:128"`
	Metadata              string    `gorm:"type:text"`
	Controller            string    `gorm:"size:128"`
	ControllerAction      string    `gorm:"size:128"`
	IsDefault             bool      `gorm:"not null;default:false;index"`
	ShowTitle             bool      `gorm:"not null;default:true"`
	Groups                []Group   `gorm:"many2many:route_groups;"`
}

type Menu struct {
	MenuID   uint   `gorm:"primaryKey"`
	MenuName string `gorm:"size:128;uniqueIndex;not null"`
	Status   Status `gorm:"size:16;not null;default:active"`
	Items    []MenuItem `gorm:"foreignKey:MenuID;constraint:-"`
}

type MenuItem struct {
	MenuItemID uint   `gorm:"primaryKey"`
	MenuID     uint   `gorm:"index;not null"`
	ParentID   *uint  `gorm:"index"`
	Parent     *MenuItem `gorm:"foreignKey:ParentID;references:MenuItemID;constraint:-"`
	Children   []MenuItem `gorm:"foreignKey:ParentID;constraint:-"`
	Name       string `gorm:"size:128;not null"`
	RouteID    *uint  `gorm:"index"`
	Route      *Route `gorm:"foreignKey:RouteID;references:RouteID;constraint:-"`
	Class      string `gorm:"size:256"`
	Target     string `gorm:"size:64"`
	Sort       int    `gorm:"not null;default:0"`
	Groups     []Group `gorm:"many2many:menu_item_groups;"`
}

type GroupKind string

const (
	GroupKindBackend  GroupKind = "backend"
	GroupKindFrontend GroupKind = "frontend"
)

type Group struct {
	GroupID  uint      `gorm:"primaryKey"`
	Name     string    `gorm:"size:128;uniqueIndex;not null"`
	ParentID *uint     `gorm:"index"`
	Kind     GroupKind `gorm:"size:16;not null;default:backend;index"`
	Status   Status    `gorm:"size:16;not null;default:active"`
	Roles    []Role    `gorm:"many2many:group_roles;"`
	Parent   *Group    `gorm:"foreignKey:ParentID;references:GroupID;constraint:-"`
}

type Role struct {
	RoleID uint   `gorm:"primaryKey"`
	Name   string `gorm:"size:128;uniqueIndex;not null"`
	Status Status `gorm:"size:16;not null;default:active"`
}

type SessionRecord struct {
	ID        string    `gorm:"primaryKey;size:128"`
	UserID    uint      `gorm:"index"`
	Data      string    `gorm:"type:text"`
	ExpiresAt time.Time `gorm:"index"`
}

type Setting struct {
	Scope   string `gorm:"primaryKey;size:128"`
	Section string `gorm:"primaryKey;size:128"`
	Data    string `gorm:"type:text;not null"`
}

type BlockType string

const (
	BlockTypeHTML      BlockType = "html"
	BlockTypeMarkdown  BlockType = "markdown"
	BlockTypeExtension BlockType = "extension"
	BlockTypeContent        BlockType = "content"
	BlockTypeLogin          BlockType = "login"
	BlockTypeMenuVertical   BlockType = "menu-vertical"
	BlockTypeMenuHorizontal BlockType = "menu-horizontal"
)

// Block assigns content to a template space for {{space "space"}} rendering.
type UserToken struct {
	UserTokenID uint       `gorm:"primaryKey"`
	UserID      uint       `gorm:"index;not null"`
	Type        string     `gorm:"size:32;index;not null"`
	Token       string     `gorm:"size:128;uniqueIndex;not null"`
	ExpiresAt   time.Time  `gorm:"index;not null"`
	UsedAt      *time.Time `gorm:"index"`
}

type Block struct {
	BlockID          uint      `gorm:"primaryKey"`
	Name             string    `gorm:"size:128;not null"`
	Type             BlockType `gorm:"size:32;not null"`
	Space            string    `gorm:"size:128;index;not null"`
	Status           Status    `gorm:"size:16;not null;default:active"`
	Sort             int       `gorm:"not null;default:0"`
	ExtensionName    string    `gorm:"size:128"`
	ExtensionBlockID string    `gorm:"size:128"`
	Metadata         string    `gorm:"type:text"`
	Groups           []Group   `gorm:"many2many:block_groups;"`
}

func All() []any {
	return []any{
		&Authenticator{},
		&User{},
		&Profile{},
		&ProfileField{},
		&UserProfile{},
		&ProfileUserFieldValue{},
		&Extension{},
		&Route{},
		&Menu{},
		&MenuItem{},
		&Group{},
		&Role{},
		&SessionRecord{},
		&Setting{},
		&UserToken{},
		&Block{},
		&Category{},
		&Tag{},
		&Item{},
		&ContentFieldGroup{},
		&ContentField{},
		&ItemFieldValue{},
		&MediaAsset{},
		&Comment{},
		&Notification{},
		&NotificationEvent{},
		&GroupAdminRoute{},
	}
}
