package oql

import (
	"fmt"
	"strings"
)

type AliasEntry struct {
	Canonical string
	Priority  int
}

type SCIMResolver struct {
	aliases        map[string]AliasEntry
	ReverseAliases map[string][]string
	FieldMeta      map[string]FieldMeta
	Warnings       []string
}

func NewSCIMResolver() *SCIMResolver {
	r := &SCIMResolver{
		aliases:        make(map[string]AliasEntry),
		ReverseAliases: make(map[string][]string),
		FieldMeta:      make(map[string]FieldMeta),
	}
	r.loadCore()
	return r
}

func (r *SCIMResolver) addAlias(alias, canonical string, priority int) {
	key := strings.ToLower(alias)
	if existing, exists := r.aliases[key]; exists && existing.Canonical != canonical {
		if priority >= existing.Priority {
			return
		}
		r.Warnings = append(r.Warnings, fmt.Sprintf("alias '%s' remapped from '%s' to '%s'", alias, existing.Canonical, canonical))
	}
	r.aliases[key] = AliasEntry{Canonical: canonical, Priority: priority}
	r.ReverseAliases[canonical] = append(r.ReverseAliases[canonical], alias)
}

func (r *SCIMResolver) loadCore() {
	r.addAlias("src_ip", "src.ip.address", 1)
	r.addAlias("source.ip", "src.ip.address", 1)
	r.addAlias("SrcAddr", "src.ip.address", 2)
	r.addAlias("src_addr", "src.ip.address", 1)
	r.addAlias("dst_ip", "dst.ip.address", 1)
	r.addAlias("dest_ip", "dst.ip.address", 1)
	r.addAlias("DstAddr", "dst.ip.address", 2)
	r.addAlias("src_port", "src.port", 1)
	r.addAlias("dst_port", "dst.port", 1)
	r.addAlias("dest_port", "dst.port", 1)
	r.addAlias("user", "actor.user.name", 1)
	r.addAlias("username", "actor.user.name", 1)
	r.addAlias("src_user", "actor.user.name", 1)
	r.addAlias("AccountName", "actor.user.name", 2)
	r.addAlias("TargetUserName", "actor.user.name", 2)
	r.addAlias("user_id", "actor.user.uid", 1)
	r.addAlias("uid", "actor.user.uid", 1)
	r.addAlias("_time", "time", 1)
	r.addAlias("timestamp", "time", 1)
	r.addAlias("@timestamp", "time", 1)
	r.addAlias("EventTime", "time", 2)
	r.addAlias("source", "metadata.source.name", 1)
	r.addAlias("sourcetype", "metadata.source.type", 1)
	r.addAlias("index", "metadata.index", 1)
	r.addAlias("host", "metadata.source.host", 1)
	r.addAlias("severity", "severity_id", 1)
	r.addAlias("level", "severity_id", 1)
	r.addAlias("action", "activity_name", 1)
	r.addAlias("eventid", "metadata.event_id", 1)
	r.addAlias("EventID", "metadata.event_id", 2)
	r.addAlias("process", "process.name", 1)
	r.addAlias("process_name", "process.name", 1)
	r.addAlias("Image", "process.file.path", 2)
	r.addAlias("CommandLine", "process.cmd_line", 2)
	r.addAlias("ParentImage", "process.parent_process.file.path", 2)
	r.addAlias("pid", "process.pid", 1)
	r.addAlias("ppid", "process.parent_process.pid", 1)
	r.addAlias("file_path", "file.path", 1)
	r.addAlias("file_name", "file.name", 1)
	r.addAlias("sha256", "file.hashes.sha256", 1)
	r.addAlias("md5", "file.hashes.md5", 1)

	r.FieldMeta["src.ip.address"] = FieldMeta{Type: FieldIP, Indexed: true, Searchable: true, DataModel: "Network"}
	r.FieldMeta["dst.ip.address"] = FieldMeta{Type: FieldIP, Indexed: true, Searchable: true, DataModel: "Network"}
	r.FieldMeta["src.port"] = FieldMeta{Type: FieldNumber, Indexed: true, DataModel: "Network"}
	r.FieldMeta["dst.port"] = FieldMeta{Type: FieldNumber, Indexed: true, DataModel: "Network"}
	r.FieldMeta["actor.user.name"] = FieldMeta{Type: FieldString, Indexed: true, Searchable: true, DataModel: "Identity"}
	r.FieldMeta["time"] = FieldMeta{Type: FieldTimestamp, Indexed: true, DataModel: "Base"}
	r.FieldMeta["severity_id"] = FieldMeta{Type: FieldNumber, Indexed: true, DataModel: "Base"}
	r.FieldMeta["activity_name"] = FieldMeta{Type: FieldString, Indexed: true, DataModel: "Base"}
	r.FieldMeta["process.name"] = FieldMeta{Type: FieldString, Indexed: true, DataModel: "Endpoint"}
	r.FieldMeta["process.cmd_line"] = FieldMeta{Type: FieldString, Searchable: true, DataModel: "Endpoint"}
	r.FieldMeta["process.pid"] = FieldMeta{Type: FieldNumber, DataModel: "Endpoint"}
	r.FieldMeta["file.path"] = FieldMeta{Type: FieldString, Indexed: true, DataModel: "Endpoint"}
	r.FieldMeta["file.hashes.sha256"] = FieldMeta{Type: FieldString, Indexed: true, DataModel: "Endpoint"}
	r.FieldMeta["metadata.source.name"] = FieldMeta{Type: FieldString, Indexed: true, DataModel: "Base"}
	r.FieldMeta["metadata.source.type"] = FieldMeta{Type: FieldString, Indexed: true, DataModel: "Base"}
	r.FieldMeta["metadata.event_id"] = FieldMeta{Type: FieldNumber, Indexed: true, DataModel: "Base"}
}

func (r *SCIMResolver) Resolve(f FieldRef) FieldRef {
	raw := f.Raw
	if raw == "" {
		raw = strings.Join(f.Parts, ".")
	}
	if _, isCanonical := r.FieldMeta[raw]; isCanonical {
		return FieldRef{Parts: strings.Split(raw, "."), Raw: raw}
	}
	key := strings.ToLower(raw)
	if entry, ok := r.aliases[key]; ok {
		return FieldRef{Parts: strings.Split(entry.Canonical, "."), Raw: raw}
	}
	return f
}

func (r *SCIMResolver) ResolveQuery(q *Query) *Query {
	out := *q
	if q.Search != nil {
		out.Search = r.resolveSearch(q.Search)
	}
	out.Commands = make([]Command, len(q.Commands))
	for i, c := range q.Commands {
		out.Commands[i] = r.resolveCmd(c)
	}
	return &out
}

func (r *SCIMResolver) resolveSearch(e SearchExpr) SearchExpr {
	if e == nil {
		return nil
	}
	switch x := e.(type) {
	case *AndExpr:
		return &AndExpr{Left: r.resolveSearch(x.Left), Right: r.resolveSearch(x.Right)}
	case *OrExpr:
		return &OrExpr{Left: r.resolveSearch(x.Left), Right: r.resolveSearch(x.Right)}
	case *NotExpr:
		return &NotExpr{Expr: r.resolveSearch(x.Expr)}
	case *CompareExpr:
		return &CompareExpr{Field: r.Resolve(x.Field), Op: x.Op, Value: x.Value}
	case *FieldExistsExpr:
		return &FieldExistsExpr{Field: r.Resolve(x.Field), Exists: x.Exists}
	default:
		return e
	}
}

func (r *SCIMResolver) resolveCmd(cmd Command) Command {
	switch c := cmd.(type) {
	case *WhereCommand:
		return &WhereCommand{Expr: r.resolveSearch(c.Expr)}
	case *StatsCommand:
		a := make([]AggExpr, len(c.Aggregations))
		for i, x := range c.Aggregations {
			a[i] = x
			if x.Field != nil {
				f := r.Resolve(*x.Field)
				a[i].Field = &f
			}
		}
		g := make([]FieldRef, len(c.GroupBy))
		for i, f := range c.GroupBy {
			g[i] = r.Resolve(f)
		}
		return &StatsCommand{Aggregations: a, GroupBy: g}
	case *TableCommand:
		f := make([]FieldRef, len(c.Fields))
		for i, x := range c.Fields {
			f[i] = r.Resolve(x)
		}
		return &TableCommand{Fields: f}
	case *SortCommand:
		s := make([]SortSpec, len(c.Specs))
		for i, x := range c.Specs {
			s[i] = SortSpec{Field: r.Resolve(x.Field), Descending: x.Descending}
		}
		return &SortCommand{Specs: s}
	case *LookupCommand:
		lc := *c
		lc.MatchField = r.Resolve(c.MatchField)
		if c.AsField != nil {
			f := r.Resolve(*c.AsField)
			lc.AsField = &f
		}
		return &lc
	default:
		return cmd
	}
}

func (r *SCIMResolver) Autocomplete(prefix string) []FieldSuggestion {
	var out []FieldSuggestion
	p := strings.ToLower(prefix)
	seen := map[string]bool{}
	for canonical, meta := range r.FieldMeta {
		if strings.HasPrefix(strings.ToLower(canonical), p) && !seen[canonical] {
			out = append(out, FieldSuggestion{Name: canonical, Type: meta.Type, DataModel: meta.DataModel, IsCanonical: true, Priority: 0})
			seen[canonical] = true
		}
	}
	for alias, entry := range r.aliases {
		if strings.HasPrefix(alias, p) && !seen[entry.Canonical] {
			meta := r.FieldMeta[entry.Canonical]
			out = append(out, FieldSuggestion{Name: alias, Type: meta.Type, DataModel: meta.DataModel, Canonical: entry.Canonical, Priority: entry.Priority})
			seen[entry.Canonical] = true
		}
	}
	return out
}

type FieldSuggestion struct {
	Name        string
	DataModel   string
	Description string
	Canonical   string
	Type        FieldType
	IsCanonical bool
	Priority    int
}
