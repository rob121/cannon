package controllers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rob121/cannon/internal/config"
)

const returnURLTTL = 24 * time.Hour

// EncodeReturn signs a relative return path for use in ?return= query values.
func EncodeReturn(site *config.SiteConfig, path string) (string, error) {
	path, ok := SafeRelativePath(path)
	if !ok {
		return "", fmt.Errorf("invalid return path")
	}
	exp := time.Now().Add(returnURLTTL).Unix()
	payload := path + "|" + strconv.FormatInt(exp, 10)
	mac := hmac.New(sha256.New, siteSecret(site))
	_, _ = mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return base64.RawURLEncoding.EncodeToString([]byte(payload + "|" + sig)), nil
}

// DecodeReturn validates and returns the signed relative path.
func DecodeReturn(site *config.SiteConfig, encoded string) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return "", fmt.Errorf("invalid return value")
	}
	parts := strings.Split(string(raw), "|")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid return value")
	}
	path, ok := SafeRelativePath(parts[0])
	if !ok {
		return "", fmt.Errorf("invalid return path")
	}
	exp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return "", fmt.Errorf("return value expired")
	}
	payload := parts[0] + "|" + parts[1]
	mac := hmac.New(sha256.New, siteSecret(site))
	_, _ = mac.Write([]byte(payload))
	expected, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", fmt.Errorf("invalid return signature")
	}
	if !hmac.Equal(expected, mac.Sum(nil)) {
		return "", fmt.Errorf("invalid return signature")
	}
	return path, nil
}

// SafeRelativePath accepts same-site relative paths only.
func SafeRelativePath(path string) (string, bool) {
	path = strings.TrimSpace(path)
	if path == "" || !strings.HasPrefix(path, "/") {
		return "", false
	}
	if strings.HasPrefix(path, "//") {
		return "", false
	}
	lower := strings.ToLower(path)
	if strings.HasPrefix(lower, "/http:") || strings.HasPrefix(lower, "/https:") {
		return "", false
	}
	if strings.Contains(path, "://") {
		return "", false
	}
	return path, true
}

func siteSecret(site *config.SiteConfig) []byte {
	sum := sha256.Sum256([]byte(site.ID + "|" + site.TmpDir + "|" + site.Host + "|cannon-return"))
	return sum[:]
}

// ReturnParam reads return from the query string or signs the fallback path.
func ReturnParam(site *config.SiteConfig, r *http.Request, fallback string) string {
	if v := strings.TrimSpace(r.URL.Query().Get("return")); v != "" {
		if path, err := DecodeReturn(site, v); err == nil {
			return path
		}
	}
	if path, ok := SafeRelativePath(fallback); ok {
		return path
	}
	return "/"
}

// AppendReturn adds a signed return query parameter to a path.
func AppendReturn(site *config.SiteConfig, basePath, returnPath string) (string, error) {
	encoded, err := EncodeReturn(site, returnPath)
	if err != nil {
		return "", err
	}
	sep := "?"
	if strings.Contains(basePath, "?") {
		sep = "&"
	}
	return basePath + sep + "return=" + encoded, nil
}
