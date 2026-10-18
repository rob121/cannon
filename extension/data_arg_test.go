package extension_test

import (
	"strings"
	"testing"

	"github.com/rob121/cannon/extension"
)

func TestDataArgKey(t *testing.T) {
	key, ok := extension.DataArgKey([]string{"bin", "--data=extensionName"})
	if !ok || key != "extensionName" {
		t.Fatalf("got %q ok=%v", key, ok)
	}
	key, ok = extension.DataArgKey([]string{"bin", "--data", "extensionVersion"})
	if !ok || key != "extensionVersion" {
		t.Fatalf("got %q ok=%v", key, ok)
	}
	_, ok = extension.DataArgKey([]string{"bin", "--site=x"})
	if ok {
		t.Fatal("expected no data arg")
	}
}

func TestInfoData(t *testing.T) {
	info := extension.Info{
		Name:          "cannon-ext-gdrive-gallery",
		Title:         "Google Drive Gallery",
		Description:   "desc",
		Version:       "0.1.4",
		UpdateURLBase: "https://example.com/x",
		AdminMenuName: "Drive Gallery",
	}
	got, err := extension.InfoData(info, "extensionName")
	if err != nil || got != "cannon-ext-gdrive-gallery" {
		t.Fatalf("extensionName=%q err=%v", got, err)
	}
	got, err = extension.InfoData(info, "extensionVersion")
	if err != nil || got != "0.1.4" {
		t.Fatalf("extensionVersion=%q err=%v", got, err)
	}
	all, err := extension.InfoData(info, "all")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(all, "extensionName=cannon-ext-gdrive-gallery") || !strings.Contains(all, "extensionVersion=0.1.4") {
		t.Fatalf("all=%q", all)
	}
	_, err = extension.InfoData(info, "nope")
	if err == nil {
		t.Fatal("expected unknown key error")
	}
}

func TestRunFromArgsDataSkipsSiteSocket(t *testing.T) {
	s := extension.New(extension.Info{
		Name:    "cannon-ext-example",
		Version: "1.2.3",
	})
	if err := s.RunFromArgs([]string{"bin", "--data=extensionVersion"}); err != nil {
		t.Fatal(err)
	}
}
