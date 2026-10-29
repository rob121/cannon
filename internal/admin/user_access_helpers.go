package admin

import (
	"net/http"

	"github.com/rob121/cannon/internal/user"
)

func actorUserID(r *http.Request) uint {
	svc, err := user.FromContext(r.Context())
	if err != nil {
		return 0
	}
	id, ok := svc.CurrentID()
	if !ok {
		return 0
	}
	return id
}
