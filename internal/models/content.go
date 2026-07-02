package models

import "time"

type ItemStatus string

const (
	ItemStatusDraft     ItemStatus = "draft"
	ItemStatusPublished ItemStatus = "published"
	ItemStatusArchived  ItemStatus = "archived"
	ItemStatusTrashed   ItemStatus = "trashed"
)

// Category is a nested content taxonomy node.
type Category struct {
	CategoryID      uint   `gorm:"primaryKey"`
	ParentID        *uint  `gorm:"index"`
	Name            string `gorm:"size:256;not null"`
	Slug            string `gorm:"size:256;uniqueIndex;not null"`
	Description     string `gorm:"type:text"`
	Image           string `gorm:"size:1024"`
	Template        string `gorm:"size:256"`
	FieldGroupID    *uint  `gorm:"index"`
	InheritSettings bool   `gorm:"not null;default:true"`
	Sort            int    `gorm:"not null;default:0"`
	Status          Status `gorm:"size:16;not null;default:active"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Groups          []Group `gorm:"many2many:category_groups;"`
}

// Tag is a reusable content label.
type Tag struct {
	TagID     uint   `gorm:"primaryKey"`
	Name      string `gorm:"size:128;not null"`
	Slug      string `gorm:"size:128;uniqueIndex;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Item is the primary CMS content unit.
type Item struct {
	ItemID          uint       `gorm:"primaryKey"`
	Title           string     `gorm:"size:512;not null"`
	Slug            string     `gorm:"size:512;uniqueIndex;not null"`
	Intro           string     `gorm:"type:text"`
	Body            string     `gorm:"type:text"`
	Status          ItemStatus `gorm:"size:16;not null;default:draft;index"`
	Featured        bool       `gorm:"not null;default:false;index"`
	PublishStart    *time.Time `gorm:"index"`
	PublishEnd      *time.Time `gorm:"index"`
	AuthorID        *uint      `gorm:"index"`
	CategoryID      *uint      `gorm:"index"`
	Image           string     `gorm:"size:1024"`
	GalleryJSON     string     `gorm:"type:text"`
	EmbedJSON       string     `gorm:"type:text"`
	AttachmentsJSON string     `gorm:"type:text"`
	MetaTitle       string     `gorm:"size:512"`
	MetaDescription string     `gorm:"type:text"`
	MetaKeywords    string     `gorm:"size:512"`
	CanonicalURL    string     `gorm:"size:1024"`
	Sort            int        `gorm:"not null;default:0;index"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Author          *User      `gorm:"foreignKey:AuthorID;references:UserID;constraint:-"`
	Category        *Category  `gorm:"foreignKey:CategoryID;references:CategoryID;constraint:-"`
	Tags            []Tag      `gorm:"many2many:item_tags;"`
	Groups          []Group    `gorm:"many2many:item_groups;"`
	FieldValues     []ItemFieldValue `gorm:"foreignKey:ItemID;constraint:-"`
}

// ContentFieldGroup groups custom fields assignable to categories.
type ContentFieldGroup struct {
	FieldGroupID uint   `gorm:"primaryKey"`
	Name         string `gorm:"size:128;uniqueIndex;not null"`
	Fields       []ContentField `gorm:"foreignKey:FieldGroupID;constraint:-"`
}

// ContentField is a custom field definition.
type ContentField struct {
	FieldID        uint   `gorm:"primaryKey"`
	FieldGroupID   uint   `gorm:"index;not null"`
	Name           string `gorm:"size:128;not null"`
	Label          string `gorm:"size:256;not null"`
	Type           string `gorm:"size:64;not null"`
	Required       bool   `gorm:"not null;default:false"`
	Sort           int    `gorm:"not null;default:0"`
	Status         Status `gorm:"size:16;not null;default:active"`
	Configuration  string `gorm:"type:text"`
}

// ItemFieldValue stores a custom field value for an item.
type ItemFieldValue struct {
	ItemID  uint   `gorm:"primaryKey"`
	FieldID uint   `gorm:"primaryKey"`
	Value   string `gorm:"type:text"`
}

// MediaAsset is an uploaded file in the media library.
type MediaAsset struct {
	MediaID   uint   `gorm:"primaryKey"`
	Name      string `gorm:"size:512;not null"`
	Path      string `gorm:"size:1024;not null;index"`
	MIME      string `gorm:"size:128"`
	Size      int64  `gorm:"not null;default:0"`
	Folder    string `gorm:"size:256;index"`
	Alt       string `gorm:"size:512"`
	Width     int    `gorm:"not null;default:0"`
	Height    int    `gorm:"not null;default:0"`
	UserID    *uint  `gorm:"index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Comment is a user comment on an item.
type Comment struct {
	CommentID   uint   `gorm:"primaryKey"`
	ItemID      uint   `gorm:"index;not null"`
	UserID      *uint  `gorm:"index"`
	AuthorName  string `gorm:"size:128"`
	AuthorEmail string `gorm:"size:256"`
	Body        string `gorm:"type:text;not null"`
	Approved    bool   `gorm:"not null;default:false;index"`
	IP          string `gorm:"size:64"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Item        *Item  `gorm:"foreignKey:ItemID;references:ItemID;constraint:-"`
	User        *User  `gorm:"foreignKey:UserID;references:UserID;constraint:-"`
}
