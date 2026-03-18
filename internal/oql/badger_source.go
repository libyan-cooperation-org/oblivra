package oql

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kingknull/oblivrashell/internal/storage"
)

// BadgerSource implements DataSource by querying the SIEM HotStore (BadgerDB).
type BadgerSource struct {
	HotStore *storage.HotStore
	TenantID string
}

func NewBadgerSource(store *storage.HotStore, tenantID string) *BadgerSource {
	if tenantID == "" {
		tenantID = "GLOBAL"
	}
	return &BadgerSource{HotStore: store, TenantID: tenantID}
}

func (s *BadgerSource) Fetch(ctx context.Context, search SearchExpr, timeRange TimeRange) ([]Row, error) {
	if s.HotStore == nil {
		return nil, fmt.Errorf("badger hot store not initialized")
	}

	prefix := []byte(fmt.Sprintf("event:%s:", s.TenantID))
	
	// Optimization: Determine if we can use a specialized index
	if ip, ok := findIPInSearch(search); ok {
		prefix = []byte(fmt.Sprintf("idx:ip:%s:%s:", s.TenantID, ip))
	} else if h, ok := findHostInSearch(search); ok {
		prefix = []byte(fmt.Sprintf("event:%s:%s:", s.TenantID, h))
	}

	var rows []Row
	// Limit to 10000 for safety in MVP, should be configurable.
	err := s.HotStore.ReverseIteratePrefix(prefix, 10000, func(key, value []byte) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var row map[string]interface{}
		if err := json.Unmarshal(value, &row); err != nil {
			return nil // Skip malformed rows
		}
		
		// Convert database.HostEvent to OQL Row (map[string]interface{})
		rows = append(rows, Row(row))
		return nil
	})

	if err != nil {
		return nil, err
	}

	return rows, nil
}

func findIPInSearch(expr SearchExpr) (string, bool) {
	if expr == nil {
		return "", false
	}
	switch e := expr.(type) {
	case *CompareExpr:
		if (e.Field.Raw == "src_ip" || e.Field.Raw == "source.ip" || e.Field.Raw == "src.ip.address") && e.Op == OpEq {
			if sv, ok := e.Value.(StringValue); ok {
				return sv.V, true
			}
		}
	case *AndExpr:
		if ip, ok := findIPInSearch(e.Left); ok {
			return ip, true
		}
		return findIPInSearch(e.Right)
	}
	return "", false
}

func findHostInSearch(expr SearchExpr) (string, bool) {
	if expr == nil {
		return "", false
	}
	switch e := expr.(type) {
	case *CompareExpr:
		if (e.Field.Raw == "host" || e.Field.Raw == "metadata.source.host") && e.Op == OpEq {
			if sv, ok := e.Value.(StringValue); ok {
				return sv.V, true
			}
		}
	case *AndExpr:
		if h, ok := findHostInSearch(e.Left); ok {
			return h, true
		}
		return findHostInSearch(e.Right)
	}
	return "", false
}
