package parsers

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/kingknull/oblivra/internal/events"
)

// Phase 98 — AWS CloudTrail JSON parser.
//
// CloudTrail delivers `{"Records": [...]}` bundles to S3; most shippers
// (and OBLIVRA's importer) unbundle to one record per line before ingest.
// This parser handles a single record object:
//
//	{"eventVersion":"1.08","eventTime":"2026-08-01T12:00:00Z",
//	 "eventSource":"signin.amazonaws.com","eventName":"ConsoleLogin",
//	 "userIdentity":{"type":"IAMUser","arn":"arn:aws:iam::111:user/alice",
//	                 "userName":"alice"},
//	 "sourceIPAddress":"203.0.113.5","awsRegion":"eu-west-1",
//	 "errorCode":"","responseElements":{"ConsoleLogin":"Success"}}
//
// Mapping: eventName → eventType (prefixed cloudtrail:), userIdentity.arn
// → user (userName preferred when present), sourceIPAddress → srcIP,
// awsRegion/eventSource/errorCode → Fields. errorCode / "Failure"
// response upgrades severity to warning so failed console logins surface
// in triage and feed the auth correlator.

type cloudTrailRecord struct {
	EventVersion string `json:"eventVersion"`
	EventTime    string `json:"eventTime"`
	EventSource  string `json:"eventSource"`
	EventName    string `json:"eventName"`
	AWSRegion    string `json:"awsRegion"`
	SourceIP     string `json:"sourceIPAddress"`
	UserAgent    string `json:"userAgent"`
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
	RecipientAcc string `json:"recipientAccountId"`
	UserIdentity struct {
		Type        string `json:"type"`
		ARN         string `json:"arn"`
		UserName    string `json:"userName"`
		AccountID   string `json:"accountId"`
		AccessKeyID string `json:"accessKeyId"`
	} `json:"userIdentity"`
	RequestParameters json.RawMessage `json:"requestParameters"`
	ResponseElements  json.RawMessage `json:"responseElements"`
}

func parseCloudTrail(raw string) (*events.Event, error) {
	var rec cloudTrailRecord
	if err := json.Unmarshal([]byte(raw), &rec); err != nil {
		return nil, errors.New("cloudtrail: " + err.Error())
	}
	if rec.EventVersion == "" || rec.EventName == "" {
		return nil, errors.New("cloudtrail: not a CloudTrail record")
	}

	user := rec.UserIdentity.UserName
	if user == "" {
		user = rec.UserIdentity.ARN
	}
	ev := &events.Event{
		Source:    events.SourceFile,
		Raw:       raw,
		EventType: "cloudtrail:" + rec.EventName,
		Severity:  events.SeverityInfo,
		Fields: map[string]string{
			"eventSource": rec.EventSource,
		},
	}
	if ts, err := time.Parse(time.RFC3339, rec.EventTime); err == nil {
		ev.Timestamp = ts
	}
	setIf := func(k, v string) {
		if v != "" {
			ev.Fields[k] = v
		}
	}
	setIf("user", user)
	setIf("userType", rec.UserIdentity.Type)
	setIf("accountId", rec.UserIdentity.AccountID)
	setIf("accessKeyId", rec.UserIdentity.AccessKeyID)
	setIf("srcIP", rec.SourceIP)
	setIf("awsRegion", rec.AWSRegion)
	setIf("userAgent", rec.UserAgent)
	setIf("errorCode", rec.ErrorCode)
	setIf("errorMessage", rec.ErrorMessage)
	setIf("recipientAccountId", rec.RecipientAcc)

	// Flatten top-level scalar requestParameters — nested objects stay
	// unexpanded (they're still queryable via Raw full-text).
	flattenScalars(rec.RequestParameters, "req.", ev.Fields)

	msg := rec.EventName + " via " + rec.EventSource
	if user != "" {
		msg += " by " + user
	}
	if rec.SourceIP != "" {
		msg += " from " + rec.SourceIP
	}
	failed := rec.ErrorCode != "" || strings.Contains(string(rec.ResponseElements), `"Failure"`)
	if failed {
		ev.Severity = events.SeverityWarn
		if rec.ErrorCode != "" {
			msg += " error=" + rec.ErrorCode
		} else {
			msg += " result=Failure"
		}
	}
	ev.Message = msg
	return ev, nil
}

// flattenScalars copies top-level string/number/bool members of a JSON
// object into fields under prefix. Silently no-ops on null/array/malformed.
func flattenScalars(rawJSON json.RawMessage, prefix string, fields map[string]string) {
	if len(rawJSON) == 0 {
		return
	}
	var m map[string]any
	if json.Unmarshal(rawJSON, &m) != nil {
		return
	}
	for k, v := range m {
		switch t := v.(type) {
		case string:
			fields[prefix+k] = t
		case float64:
			fields[prefix+k] = strconv.FormatFloat(t, 'f', -1, 64)
		case bool:
			fields[prefix+k] = strconv.FormatBool(t)
		}
	}
}
