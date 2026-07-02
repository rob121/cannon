package auth

import (
	"context"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

// DefaultProviders lists Goth providers seeded at install.
var DefaultProviders = []string{
	"amazon",
	"apple",
	"auth0",
	"azureadv2",
	"battlenet",
	"bitbucket",
	"box",
	"digitalocean",
	"discord",
	"dropbox",
	"facebook",
	"fitbit",
	"gitea",
	"github",
	"gitlab",
	"google",
	"heroku",
	"instagram",
	"intercom",
	"kakao",
	"lastfm",
	"linkedin",
	"line",
	"mastodon",
	"meetup",
	"microsoftonline",
	"naver",
	"nextcloud",
	"okta",
	"openid",
	"paypal",
	"salesforce",
	"seatalk",
	"shopify",
	"slack",
	"soundcloud",
	"spotify",
	"steam",
	"strava",
	"stripe",
	"tiktok",
	"twitch",
	"twitter",
	"typetalk",
	"uber",
	"vk",
	"wecom",
	"xero",
	"yahoo",
	"yammer",
	"yandex",
	"zoom",
}

// SeedAuthenticators inserts inactive provider rows.
func SeedAuthenticators(db *gorm.DB) error {
	for _, name := range DefaultProviders {
		var existing models.Authenticator
		err := db.Where("name = ?", name).First(&existing).Error
		if err == nil {
			continue
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}
		cfg := DefaultConfigJSON(name)
		row := models.Authenticator{
			Name:          name,
			Status:        models.StatusInactive,
			Configuration: cfg,
		}
		if err := db.Create(&row).Error; err != nil {
			return err
		}
	}

	var local models.Authenticator
	if err := db.Where("name = ?", "local").First(&local).Error; err == gorm.ErrRecordNotFound {
		local = models.Authenticator{
			Name:          "local",
			Status:        models.StatusActive,
			Configuration: DefaultConfigJSON("local"),
		}
		if err := db.Create(&local).Error; err != nil {
			return err
		}
	}
	return nil
}

// ActiveProviders returns enabled authenticators for a site.
func ActiveProviders(ctx context.Context) ([]models.Authenticator, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	var rows []models.Authenticator
	err = db.Where("status = ?", models.StatusActive).Find(&rows).Error
	return rows, err
}
