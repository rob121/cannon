package extension

import (
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
)

type configurationRow struct {
	Section string `gorm:"primaryKey;size:128"`
	Data    string `gorm:"type:text;not null"`
}

// DBConfigurationStore persists extension settings in an extension-owned table.
func DBConfigurationStore(db *gorm.DB, table string) ConfigurationStore {
	return &dbConfigurationStore{db: db, table: table}
}

type dbConfigurationStore struct {
	db    *gorm.DB
	table string
}

func (s *dbConfigurationStore) Load(section string) (map[string]any, error) {
	var row configurationRow
	err := s.db.Table(s.table).Where("section = ?", section).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}
	if row.Data == "" {
		return map[string]any{}, nil
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(row.Data), &data); err != nil {
		return nil, fmt.Errorf("decode configuration: %w", err)
	}
	return data, nil
}

func (s *dbConfigurationStore) Save(section string, data map[string]any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	row := configurationRow{Section: section, Data: string(raw)}
	return s.db.Table(s.table).Save(&row).Error
}

// MigrateConfigurationStore creates the configuration table for an extension.
func MigrateConfigurationStore(db *gorm.DB, table string) error {
	return db.Table(table).AutoMigrate(&configurationRow{})
}
