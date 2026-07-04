package user

import (
	"testing"

	"github.com/rob121/cannon/internal/models"
)

func TestResolveAvatarPrefersUploaded(t *testing.T) {
	u := &models.User{
		AvatarURL:    "/assets/media/avatars/user-1.png",
		SSOAvatarURL: "https://provider.example/photo.jpg",
	}
	if got := ResolveAvatar(u); got != u.AvatarURL {
		t.Fatalf("got %q", got)
	}
}

func TestResolveAvatarUsesSSOFallback(t *testing.T) {
	u := &models.User{SSOAvatarURL: "https://provider.example/photo.jpg"}
	if got := ResolveAvatar(u); got != u.SSOAvatarURL {
		t.Fatalf("got %q", got)
	}
}
