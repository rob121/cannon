package blocks

import (
	"errors"
	"sort"
	"strings"

	"github.com/rob121/cannon/internal/models"
	"gorm.io/gorm"
)

// ErrNoSpaces is returned when a block would be saved with no space memberships.
var ErrNoSpaces = errors.New("at least one space is required")

// NormalizeSpaces trims, drops empties, and de-duplicates while preserving order.
func NormalizeSpaces(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, raw := range values {
		space := strings.TrimSpace(raw)
		if space == "" {
			continue
		}
		if _, ok := seen[space]; ok {
			continue
		}
		seen[space] = struct{}{}
		out = append(out, space)
	}
	return out
}

// SpaceNames returns the space strings assigned to a block.
func SpaceNames(row models.Block) []string {
	if len(row.Spaces) > 0 {
		names := make([]string, 0, len(row.Spaces))
		for _, membership := range row.Spaces {
			if space := strings.TrimSpace(membership.Space); space != "" {
				names = append(names, space)
			}
		}
		if len(names) > 0 {
			return names
		}
	}
	if space := strings.TrimSpace(row.Space); space != "" {
		return []string{space}
	}
	return nil
}

// PrimarySpace returns the first assigned space (legacy column / display).
func PrimarySpace(row models.Block) string {
	names := SpaceNames(row)
	if len(names) == 0 {
		return ""
	}
	return names[0]
}

// ReplaceBlockSpaces replaces memberships for a block. Existing sorts are kept
// when a space is retained; new spaces append at the end of that space's order.
func ReplaceBlockSpaces(db *gorm.DB, blockID uint, spaces []string) error {
	spaces = NormalizeSpaces(spaces)
	if len(spaces) == 0 {
		return ErrNoSpaces
	}

	var existing []models.BlockSpace
	if err := db.Where("block_id = ?", blockID).Find(&existing).Error; err != nil {
		return err
	}
	bySpace := make(map[string]models.BlockSpace, len(existing))
	for _, row := range existing {
		bySpace[row.Space] = row
	}

	keep := make(map[string]struct{}, len(spaces))
	for _, space := range spaces {
		keep[space] = struct{}{}
		if _, ok := bySpace[space]; ok {
			continue
		}
		nextSort, err := nextSortInSpace(db, space)
		if err != nil {
			return err
		}
		if err := db.Create(&models.BlockSpace{
			BlockID: blockID,
			Space:   space,
			Sort:    nextSort,
		}).Error; err != nil {
			return err
		}
	}

	for space, row := range bySpace {
		if _, ok := keep[space]; ok {
			continue
		}
		if err := db.Delete(&models.BlockSpace{}, "block_space_id = ?", row.BlockSpaceID).Error; err != nil {
			return err
		}
	}

	primary := spaces[0]
	return db.Model(&models.Block{}).Where("block_id = ?", blockID).
		Updates(map[string]any{"space": primary}).Error
}

func nextSortInSpace(db *gorm.DB, space string) (int, error) {
	var maxSort *int
	err := db.Model(&models.BlockSpace{}).
		Where("space = ?", space).
		Select("MAX(sort)").
		Scan(&maxSort).Error
	if err != nil {
		return 0, err
	}
	if maxSort == nil {
		return 0, nil
	}
	return *maxSort + 1, nil
}

// DistinctSpaces returns unique space names from block_spaces (with legacy fallback).
func DistinctSpaces(db *gorm.DB) ([]string, error) {
	var spaces []string
	err := db.Model(&models.BlockSpace{}).Distinct().Order("space asc").Pluck("space", &spaces).Error
	if err != nil {
		return nil, err
	}
	if len(spaces) > 0 {
		return spaces, nil
	}
	// Legacy DBs mid-migrate may still only have blocks.space populated.
	err = db.Model(&models.Block{}).Distinct().Order("space asc").Pluck("space", &spaces).Error
	if err != nil {
		return nil, err
	}
	out := NormalizeSpaces(spaces)
	sort.Strings(out)
	return out, nil
}
