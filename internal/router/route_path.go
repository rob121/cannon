package router

// BuiltinRoutePath returns the default path for a built-in controller action.
func BuiltinRoutePath(controller, action string) string {
	for _, br := range builtinRouteDefs {
		if br.Controller == controller && br.ControllerAction == action {
			return br.Path
		}
	}
	return ""
}
