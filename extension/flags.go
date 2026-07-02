package extension

import (
	"fmt"
	"strings"
)

// Flags holds Cannon startup arguments passed to extension processes.
type Flags struct {
	SiteID     string
	SocketPath string
}

// ParseFlags reads --site= and --socket= from process arguments.
func ParseFlags(args []string) (Flags, error) {
	flags := Flags{}
	for _, arg := range args[1:] {
		switch {
		case strings.HasPrefix(arg, "--site="):
			flags.SiteID = strings.TrimPrefix(arg, "--site=")
		case strings.HasPrefix(arg, "--socket="):
			flags.SocketPath = strings.TrimPrefix(arg, "--socket=")
		}
	}
	if flags.SiteID == "" {
		return Flags{}, fmt.Errorf("missing --site argument")
	}
	if flags.SocketPath == "" {
		return Flags{}, fmt.Errorf("missing --socket argument")
	}
	return flags, nil
}
