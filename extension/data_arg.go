package extension

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// DataArgKey extracts --data=KEY (or --data KEY) from args.
// ok is false when the flag is not present.
func DataArgKey(args []string) (key string, ok bool) {
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--data":
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				return args[i+1], true
			}
			return "", true
		case strings.HasPrefix(arg, "--data="):
			return strings.TrimPrefix(arg, "--data="), true
		}
	}
	return "", false
}

// InfoData returns the value for a metadata key drawn from Info
// (the usual main.go constants: extensionName, extensionVersion, …).
// An empty key or "all"/"list" returns every known key as "key=value" lines.
func InfoData(info Info, key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" || strings.EqualFold(key, "all") || strings.EqualFold(key, "list") {
		return formatAllInfoData(info), nil
	}
	val, ok := lookupInfoData(info, key)
	if !ok {
		keys := infoDataKeys()
		return "", fmt.Errorf("unknown --data key %q (want one of: %s)", key, strings.Join(keys, ", "))
	}
	return val, nil
}

// WriteDataArg handles --data=… from args using info.
// When the flag is present it writes the value to w and returns handled=true.
// Site/socket flags are not required.
func WriteDataArg(info Info, args []string, w io.Writer) (handled bool, err error) {
	key, ok := DataArgKey(args)
	if !ok {
		return false, nil
	}
	out, err := InfoData(info, key)
	if err != nil {
		return true, err
	}
	_, err = fmt.Fprintln(w, out)
	return true, err
}

func lookupInfoData(info Info, key string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "extensionname", "name":
		return info.Name, true
	case "extensiontitle", "title":
		return info.Title, true
	case "extensiondescription", "description":
		return info.Description, true
	case "extensionversion", "version":
		return info.Version, true
	case "updateurlbase", "update_url_base":
		return info.UpdateURLBase, true
	case "defaultadminmenu", "adminmenuname", "admin_menu_name":
		return info.AdminMenuName, true
	default:
		return "", false
	}
}

func infoDataKeys() []string {
	return []string{
		"extensionName",
		"extensionTitle",
		"extensionDescription",
		"extensionVersion",
		"updateURLBase",
		"defaultAdminMenu",
	}
}

func formatAllInfoData(info Info) string {
	keys := infoDataKeys()
	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		val, _ := lookupInfoData(info, k)
		lines = append(lines, k+"="+val)
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}
