package oql

const (
	OQLv1             = 1
	CurrentOQLVersion = OQLv1
)

type VersionedQuery struct {
	Version int    `json:"oql_version" yaml:"oql_version"`
	Query   string `json:"query" yaml:"query"`
}

func Migrate(vq VersionedQuery) (VersionedQuery, []string) {
	if vq.Version == 0 {
		vq.Version = CurrentOQLVersion
	}
	if vq.Version == CurrentOQLVersion {
		return vq, nil
	}
	return VersionedQuery{Version: CurrentOQLVersion, Query: vq.Query}, nil
}

func ParseVersioned(vq VersionedQuery, macros map[string]MacroDef) (*Query, []string, error) {
	migrated, warnings := Migrate(vq)
	ast, err := Parse(migrated.Query, macros)
	if err != nil {
		return nil, warnings, err
	}
	ast.Version = migrated.Version
	return ast, warnings, nil
}
