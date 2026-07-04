package security

// Permission describes one registered capability.
type Permission struct {
	ID          string
	DisplayName string
	Description string
	Category    string
	Dangerous   bool
}

// RoleDef describes a default role registered in code.
type RoleDef struct {
	Name        string
	Description string
	SystemRole  bool
	Inherits    []string // parent role names
	Permissions []string
}

// PolicyDef describes a future ABAC policy registered in code.
type PolicyDef struct {
	ID          string
	Description string
}
