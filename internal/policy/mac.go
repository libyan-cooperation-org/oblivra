package policy

import (
	"github.com/kingknull/oblivrashell/internal/auth"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// MACEngine strictly enforces Mandatory Access Control levels
type MACEngine struct {
	log *logger.Logger
}

func NewMACEngine(log *logger.Logger) *MACEngine {
	return &MACEngine{log: log}
}

// Evaluate checks if the subject (user/actor) has the required clearance to access the object
// This implements a standard "no read up" hierarchical policy constraint.
func (m *MACEngine) Evaluate(subjectClearance auth.Clearance, requiredClearance auth.Clearance, objectName string) bool {
	if subjectClearance >= requiredClearance {
		m.log.Debug("[MAC] Access GRANTED to '%s': Subject Clearance (%d) >= Required (%d)", objectName, subjectClearance, requiredClearance)
		return true
	}

	m.log.Warn("[MAC] Access DENIED to '%s': Subject Clearance (%d) < Required (%d)", objectName, subjectClearance, requiredClearance)
	return false
}

// clearanceNames wraps the int enum for readability in logs
var clearanceNames = map[auth.Clearance]string{
	auth.ClearanceUnclassified: "Unclassified",
	auth.ClearanceConfidential: "Confidential",
	auth.ClearanceSecret:       "Secret",
	auth.ClearanceTopSecret:    "Top Secret",
}

func GetClearanceName(c auth.Clearance) string {
	if name, ok := clearanceNames[c]; ok {
		return name
	}
	return "Unknown"
}
