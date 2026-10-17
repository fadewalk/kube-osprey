package common

import "time"

type Verb string

const (
	VerbGet    Verb = "get"
	VerbCreate Verb = "create"
	VerbUpdate Verb = "update"
	VerbDelete Verb = "delete"
	VerbLog    Verb = "log"
	VerbExec   Verb = "exec"
)

type Role struct {
	Name        string   `yaml:"name" json:"name"`
	Description string   `yaml:"description" json:"-"`
	Clusters    []string `yaml:"clusters" json:"clusters"`
	Resources   []string `yaml:"resources" json:"resources"`
	Namespaces  []string `yaml:"namespaces" json:"namespaces"`
	Verbs       []string `yaml:"verbs" json:"verbs"`
}

type RoleMapping struct {
	Name       string   `yaml:"name" json:"name"`
	Users      []string `yaml:"users,omitempty" json:"users,omitempty"`
	OIDCGroups []string `yaml:"oidcGroups,omitempty" json:"oidcGroups,omitempty"`

	// Expiry is when this mapping stops applying. It is populated from
	// DB-backed assignments with an expiry; declarative config never sets it.
	Expiry *time.Time `yaml:"-" json:"-"`
}

type RolesConfig struct {
	Roles       []Role        `yaml:"roles"`
	RoleMapping []RoleMapping `yaml:"roleMapping"`
}
