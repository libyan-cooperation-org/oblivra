package detection

// Mitigation of rigid string lookups by centralizing MITRE ATT&CK schema.
// This allows the frontend to dynamically group triggers by standard T-Codes.

var Tactics = map[string]string{
	"TA0001": "Initial Access",
	"TA0002": "Execution",
	"TA0003": "Persistence",
	"TA0004": "Privilege Escalation",
	"TA0005": "Defense Evasion",
	"TA0006": "Credential Access",
	"TA0007": "Discovery",
	"TA0008": "Lateral Movement",
	"TA0009": "Collection",
	"TA0011": "Command and Control",
	"TA0010": "Exfiltration",
	"TA0040": "Impact",
}

var Techniques = map[string]string{
	"T1078": "Valid Accounts",
	"T1110": "Brute Force",
	"T1059": "Command and Scripting Interpreter",
	"T1098": "Account Manipulation",
	"T1562": "Impair Defenses",
	"T1046": "Network Service Discovery",
	"T1021": "Remote Services",
	"T1552": "Unsecured Credentials",
	"T1071": "Application Layer Protocol",
	"T1486": "Data Encrypted for Impact",
}

// GetTacticName returns the human-readable text for a MITRE Tactic ID (e.g. TA0001)
func GetTacticName(tacticID string) string {
	if name, ok := Tactics[tacticID]; ok {
		return name
	}
	return "Unknown Tactic"
}

// GetTechniqueName returns the human-readable text for a MITRE Technique ID (e.g. T1110)
func GetTechniqueName(techniqueID string) string {
	if name, ok := Techniques[techniqueID]; ok {
		return name
	}
	return "Unknown Technique"
}
