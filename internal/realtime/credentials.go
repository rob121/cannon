package realtime

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/centrifugal/centrifuge"
	"github.com/rob121/cannon/internal/security"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/user"
)

// ConnInfo is stored in Centrifuge connection credentials.
type ConnInfo struct {
	SiteID string `json:"site_id"`
	Admin  bool   `json:"admin"`
}

func connInfoFromRequest(r *http.Request) (ConnInfo, string, error) {
	site, err := sites.FromContext(r.Context())
	if err != nil {
		return ConnInfo{}, "", err
	}
	svc, err := user.FromContext(r.Context())
	if err != nil {
		return ConnInfo{}, "", err
	}
	visitorID, err := svc.AnalyticsVisitorID()
	if err != nil {
		return ConnInfo{}, "", err
	}
	userID := "anon:" + visitorID
	admin := false
	if uid, ok := svc.CurrentID(); ok {
		userID = "user:" + strconv.FormatUint(uint64(uid), 10)
		if ok, err := security.Can(r.Context(), uid, security.PermAdminAccess); err == nil && ok {
			admin = true
		}
	}
	info := ConnInfo{SiteID: site.ID, Admin: admin}
	raw, err := json.Marshal(info)
	if err != nil {
		return ConnInfo{}, "", err
	}
	_ = raw
	return info, userID, nil
}

// CredentialsMiddleware injects Centrifuge credentials from the Cannon session.
func CredentialsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info, userID, err := connInfoFromRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		raw, err := json.Marshal(info)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ctx := centrifuge.SetCredentials(r.Context(), &centrifuge.Credentials{
			UserID: userID,
			Info:   raw,
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func parseConnInfo(client *centrifuge.Client) (ConnInfo, bool) {
	cred, ok := centrifuge.GetCredentials(client.Context())
	if !ok || cred == nil || len(cred.Info) == 0 {
		return ConnInfo{}, false
	}
	var info ConnInfo
	if err := json.Unmarshal(cred.Info, &info); err != nil {
		return ConnInfo{}, false
	}
	return info, info.SiteID != ""
}
