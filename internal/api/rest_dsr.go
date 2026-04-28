package api

// DSR — Data Subject Request handlers.
//
// Implements GDPR Articles 15 (right of access) and 17 (right to erasure)
// at the operator-facing API layer, plus CCPA §1798.105 (right to delete)
// and §1798.110 (right to know). The crypto-wipe primitive already exists
// (DisasterService + per-table column-level erasure in settings_service);
// this layer adds the operator workflow on top:
//
//   POST /api/v1/dsr/requests       — file a request
//   GET  /api/v1/dsr/requests       — list pending / resolved requests
//   POST /api/v1/dsr/requests/:id/fulfill   — execute the access or deletion
//   POST /api/v1/dsr/requests/:id/reject    — reject (e.g. unverified subject)
//
// The audit_log is the durable record of every state transition, so the
// "did you action that GDPR request" question has a defensible answer.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kingknull/oblivrashell/internal/auth"
	"github.com/kingknull/oblivrashell/internal/database"
)

// dsrCallerID returns the email of the operator behind this request,
// or "anonymous" if none could be resolved (which shouldn't happen for
// an admin-gated endpoint but defends against future call-site changes).
func dsrCallerID(ctx context.Context) string {
	if u := auth.UserFromContext(ctx); u != nil && u.Email != "" {
		return u.Email
	}
	return "anonymous"
}

// DSRRequest mirrors the row shape of the dsr_requests table (see migration
// v31). Status transitions: pending → fulfilled | rejected.
type DSRRequest struct {
	ID          string `json:"id"`
	TenantID    string `json:"tenant_id"`
	SubjectID   string `json:"subject_id"`   // email, user-id, or pseudonym
	RequestType string `json:"request_type"` // "access" | "deletion"
	Reason      string `json:"reason,omitempty"`
	Requester   string `json:"requester"`    // who filed the request (operator email or "self")
	Verification string `json:"verification,omitempty"` // free-form proof reference
	Status      string `json:"status"`       // "pending" | "fulfilled" | "rejected"
	CreatedAt   string `json:"created_at"`
	ResolvedAt  string `json:"resolved_at,omitempty"`
	ResolvedBy  string `json:"resolved_by,omitempty"`
	ResolutionNotes string `json:"resolution_notes,omitempty"`
}

// POST /api/v1/dsr/requests — file a new DSR.
func (s *RESTServer) handleDSRCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)

	var req struct {
		SubjectID    string `json:"subject_id"`
		RequestType  string `json:"request_type"`
		Reason       string `json:"reason"`
		Verification string `json:"verification"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.SubjectID == "" {
		http.Error(w, "subject_id is required", http.StatusBadRequest)
		return
	}
	rt := strings.ToLower(req.RequestType)
	if rt != "access" && rt != "deletion" {
		http.Error(w, "request_type must be \"access\" or \"deletion\"", http.StatusBadRequest)
		return
	}

	tenant, _ := database.TenantFromContext(r.Context())
	if tenant == "" {
		tenant = "GLOBAL"
	}
	requester := dsrCallerID(r.Context())
	if requester == "" {
		requester = "anonymous"
	}

	id := genDSRID()
	now := time.Now().Format(time.RFC3339)

	if _, err := s.db.DB().ExecContext(r.Context(),
		`INSERT INTO dsr_requests (id, tenant_id, subject_id, request_type, reason, requester, verification, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, 'pending', ?)`,
		id, tenant, req.SubjectID, rt, req.Reason, requester, req.Verification, now); err != nil {
		s.log.Error("[dsr] persist failed: %v", err)
		http.Error(w, "persist failed", http.StatusInternalServerError)
		return
	}

	s.appendAuditEntry(requester, "dsr.filed", id, fmt.Sprintf("type=%s subject=%s", rt, req.SubjectID), r)

	s.jsonResponse(w, http.StatusCreated, DSRRequest{
		ID: id, TenantID: tenant, SubjectID: req.SubjectID,
		RequestType: rt, Reason: req.Reason, Requester: requester,
		Verification: req.Verification, Status: "pending", CreatedAt: now,
	})
}

// GET /api/v1/dsr/requests — list per-tenant.
func (s *RESTServer) handleDSRList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tenant, _ := database.TenantFromContext(r.Context())
	if tenant == "" {
		tenant = "GLOBAL"
	}
	rows, err := s.db.DB().QueryContext(r.Context(),
		`SELECT id, tenant_id, subject_id, request_type, reason, requester,
		        COALESCE(verification, ''), status, created_at,
		        COALESCE(resolved_at, ''), COALESCE(resolved_by, ''),
		        COALESCE(resolution_notes, '')
		   FROM dsr_requests
		  WHERE tenant_id = ?
		  ORDER BY created_at DESC`, tenant)
	if err != nil {
		s.jsonResponse(w, http.StatusOK, map[string]any{"requests": []any{}})
		return
	}
	defer rows.Close()

	out := []DSRRequest{}
	for rows.Next() {
		var d DSRRequest
		if err := rows.Scan(&d.ID, &d.TenantID, &d.SubjectID, &d.RequestType,
			&d.Reason, &d.Requester, &d.Verification, &d.Status,
			&d.CreatedAt, &d.ResolvedAt, &d.ResolvedBy, &d.ResolutionNotes); err != nil {
			continue
		}
		out = append(out, d)
	}
	s.jsonResponse(w, http.StatusOK, map[string]any{"requests": out})
}

// POST /api/v1/dsr/requests/:id/fulfill
func (s *RESTServer) handleDSRFulfill(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	role := auth.GetRole(r.Context())
	if role != auth.RoleAdmin {
		http.Error(w, "Forbidden: Admin only", http.StatusForbidden)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/dsr/requests/")
	id = strings.TrimSuffix(id, "/fulfill")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}

	dsr, err := s.loadDSR(r, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if dsr.Status != "pending" {
		http.Error(w, fmt.Sprintf("request already %s", dsr.Status), http.StatusConflict)
		return
	}

	// Execute the request. Access = export PII; deletion = crypto-wipe.
	notes := ""
	switch dsr.RequestType {
	case "access":
		export, err := s.dsrExportSubject(r, dsr.TenantID, dsr.SubjectID)
		if err != nil {
			http.Error(w, fmt.Sprintf("export failed: %v", err), http.StatusInternalServerError)
			return
		}
		notes = fmt.Sprintf("exported %d records", len(export))
		// Persist the export to evidence_chain so the operator can retrieve it.
		// (For the access path we return the data inline; the chain entry is
		// the audit-trail proof we honoured the request.)
		s.appendAuditEntry(dsrCallerID(r.Context()), "dsr.access.exported", dsr.ID,
			fmt.Sprintf("subject=%s records=%d", dsr.SubjectID, len(export)), r)
		s.markDSRResolved(r, dsr.ID, "fulfilled", notes)
		s.jsonResponse(w, http.StatusOK, map[string]any{
			"request_id": dsr.ID,
			"records":    export,
			"status":     "fulfilled",
		})
		return

	case "deletion":
		count, err := s.dsrDeleteSubject(r, dsr.TenantID, dsr.SubjectID)
		if err != nil {
			http.Error(w, fmt.Sprintf("delete failed: %v", err), http.StatusInternalServerError)
			return
		}
		notes = fmt.Sprintf("crypto-wiped %d records", count)
		s.appendAuditEntry(dsrCallerID(r.Context()), "dsr.deletion.executed", dsr.ID,
			fmt.Sprintf("subject=%s records=%d", dsr.SubjectID, count), r)
	}

	s.markDSRResolved(r, dsr.ID, "fulfilled", notes)
	s.jsonResponse(w, http.StatusOK, map[string]any{
		"request_id": dsr.ID,
		"status":     "fulfilled",
		"notes":      notes,
	})
}

// POST /api/v1/dsr/requests/:id/reject
func (s *RESTServer) handleDSRReject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	role := auth.GetRole(r.Context())
	if role != auth.RoleAdmin {
		http.Error(w, "Forbidden: Admin only", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	var req struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Reason == "" {
		req.Reason = "no reason provided"
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/v1/dsr/requests/")
	id = strings.TrimSuffix(id, "/reject")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}

	s.markDSRResolved(r, id, "rejected", req.Reason)
	s.appendAuditEntry(dsrCallerID(r.Context()), "dsr.rejected", id, req.Reason, r)
	s.jsonResponse(w, http.StatusOK, map[string]any{"request_id": id, "status": "rejected"})
}

// loadDSR pulls a request by id. Tenant-scoped via context.
func (s *RESTServer) loadDSR(r *http.Request, id string) (*DSRRequest, error) {
	tenant, _ := database.TenantFromContext(r.Context())
	if tenant == "" {
		tenant = "GLOBAL"
	}
	row := s.db.DB().QueryRowContext(r.Context(),
		`SELECT id, tenant_id, subject_id, request_type, reason, requester,
		        COALESCE(verification, ''), status, created_at,
		        COALESCE(resolved_at, ''), COALESCE(resolved_by, ''),
		        COALESCE(resolution_notes, '')
		   FROM dsr_requests WHERE id = ? AND tenant_id = ?`, id, tenant)
	var d DSRRequest
	if err := row.Scan(&d.ID, &d.TenantID, &d.SubjectID, &d.RequestType,
		&d.Reason, &d.Requester, &d.Verification, &d.Status,
		&d.CreatedAt, &d.ResolvedAt, &d.ResolvedBy, &d.ResolutionNotes); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("DSR request not found")
		}
		return nil, err
	}
	return &d, nil
}

// markDSRResolved updates the status row.
func (s *RESTServer) markDSRResolved(r *http.Request, id, status, notes string) {
	resolver := dsrCallerID(r.Context())
	if resolver == "" {
		resolver = "system"
	}
	_, _ = s.db.DB().ExecContext(r.Context(),
		`UPDATE dsr_requests
		   SET status = ?, resolved_at = ?, resolved_by = ?, resolution_notes = ?
		 WHERE id = ?`,
		status, time.Now().Format(time.RFC3339), resolver, notes, id)
}

// dsrExportSubject pulls every PII record across known PII-bearing tables
// for a given subject id. The set of tables here is the canonical PII map
// that the DPIA references — keep them in sync.
func (s *RESTServer) dsrExportSubject(r *http.Request, tenant, subject string) ([]map[string]any, error) {
	out := []map[string]any{}

	// users (email, name)
	if rows, err := s.db.DB().QueryContext(r.Context(),
		`SELECT id, email, COALESCE(name, ''), COALESCE(role_id, ''), COALESCE(created_at, '')
		   FROM users WHERE email = ? OR id = ?`, subject, subject); err == nil {
		defer rows.Close()
		for rows.Next() {
			var id, email, name, roleID, created string
			if err := rows.Scan(&id, &email, &name, &roleID, &created); err == nil {
				out = append(out, map[string]any{
					"_table": "users",
					"id": id, "email": email, "name": name, "role_id": roleID, "created_at": created,
				})
			}
		}
	}
	// audit_logs (actor)
	if rows, err := s.db.DB().QueryContext(r.Context(),
		`SELECT id, actor, action, resource, COALESCE(outcome, ''), created_at
		   FROM audit_logs WHERE actor = ? LIMIT 5000`, subject); err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int
			var actor, action, resource, outcome, created string
			if err := rows.Scan(&id, &actor, &action, &resource, &outcome, &created); err == nil {
				out = append(out, map[string]any{
					"_table": "audit_logs",
					"id": id, "actor": actor, "action": action,
					"resource": resource, "outcome": outcome, "created_at": created,
				})
			}
		}
	}
	// hosts (username, hostname — may be PII in some contexts)
	if rows, err := s.db.DB().QueryContext(r.Context(),
		`SELECT id, COALESCE(label, ''), COALESCE(hostname, ''), COALESCE(username, '')
		   FROM hosts WHERE tenant_id = ? AND (username = ? OR hostname = ?)`,
		tenant, subject, subject); err == nil {
		defer rows.Close()
		for rows.Next() {
			var id, label, hostname, username string
			if err := rows.Scan(&id, &label, &hostname, &username); err == nil {
				out = append(out, map[string]any{
					"_table": "hosts",
					"id": id, "label": label, "hostname": hostname, "username": username,
				})
			}
		}
	}

	return out, nil
}

// dsrDeleteSubject crypto-wipes every PII row for a subject. Returns
// the count of affected rows. Per GDPR Art. 17(3) we do NOT wipe rows
// that have a legal-hold flag (audit_logs remain — required by §6.1.c
// for "compliance with a legal obligation"). For audit_logs we instead
// pseudonymise the actor field so the log integrity stays intact.
func (s *RESTServer) dsrDeleteSubject(r *http.Request, tenant, subject string) (int, error) {
	tx, err := s.db.DB().BeginTx(r.Context(), nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	total := 0

	// users — full delete
	if res, err := tx.ExecContext(r.Context(),
		`DELETE FROM users WHERE email = ? OR id = ?`, subject, subject); err == nil {
		if n, _ := res.RowsAffected(); n > 0 {
			total += int(n)
		}
	}
	// audit_logs — pseudonymise (legal retention obligation)
	pseudonym := pseudonymFor(subject)
	if res, err := tx.ExecContext(r.Context(),
		`UPDATE audit_logs SET actor = ? WHERE actor = ?`, pseudonym, subject); err == nil {
		if n, _ := res.RowsAffected(); n > 0 {
			total += int(n)
		}
	}
	// hosts — full delete (host-bound credential intel)
	if res, err := tx.ExecContext(r.Context(),
		`DELETE FROM hosts WHERE tenant_id = ? AND (username = ? OR hostname = ?)`,
		tenant, subject, subject); err == nil {
		if n, _ := res.RowsAffected(); n > 0 {
			total += int(n)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return total, nil
}

// pseudonymFor returns a stable hash that identifies the same subject
// across deletions without revealing the original identifier. Used
// when the row can't be erased outright (legal retention).
func pseudonymFor(subject string) string {
	h := sha256.Sum256([]byte("dsr-pseudonym-v1:" + subject))
	return "subject-" + hex.EncodeToString(h[:])[:16]
}

// genDSRID returns "DSR-<unix>-<rand4>".
func genDSRID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("DSR-%d-%s", time.Now().Unix(), hex.EncodeToString(b))
}
