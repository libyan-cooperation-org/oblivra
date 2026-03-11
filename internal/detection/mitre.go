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
	// Initial Access
	"T1078": "Valid Accounts",
	"T1190": "Exploit Public-Facing Application",
	"T1133": "External Remote Services",
	"T1566": "Phishing",
	// Execution
	"T1059": "Command and Scripting Interpreter",
	"T1053": "Scheduled Task/Job",
	"T1047": "Windows Management Instrumentation",
	"T1203": "Exploitation for Client Execution",
	// Persistence
	"T1098": "Account Manipulation",
	"T1136": "Create Account",
	"T1543": "Create or Modify System Process",
	// Privilege Escalation
	"T1548": "Abuse Elevation Control Mechanism",
	"T1068": "Exploitation for Privilege Escalation",
	"T1134": "Access Token Manipulation",
	"T1574": "Hijack Execution Flow",
	// Defense Evasion
	"T1562": "Impair Defenses",
	"T1070": "Indicator Removal",
	"T1027": "Obfuscated Files or Information",
	"T1036": "Masquerading",
	"T1112": "Modify Registry",
	// Credential Access
	"T1110": "Brute Force",
	"T1003": "OS Credential Dumping",
	"T1555": "Credentials from Password Stores",
	"T1558": "Steal or Forge Kerberos Tickets",
	"T1552": "Unsecured Credentials",
	// Discovery
	"T1087": "Account Discovery",
	"T1069": "Permission Groups Discovery",
	"T1018": "Remote System Discovery",
	"T1046": "Network Service Discovery",
	// Lateral Movement
	"T1021": "Remote Services",
	"T1210": "Exploitation of Remote Services",
	"T1563": "Remote Service Session Hijacking",
	"T1080": "Taint Shared Content",
	// Collection
	"T1560": "Archive Collected Data",
	"T1074": "Data Staged",
	"T1005": "Data from Local System",
	// Exfiltration
	"T1048": "Exfiltration Over Alternative Protocol",
	"T1041": "Exfiltration Over C2 Channel",
	"T1567": "Exfiltration Over Web Service",
	// Command and Control
	"T1071": "Application Layer Protocol",
	"T1105": "Ingress Tool Transfer",
	"T1572": "Protocol Tunneling",
	// Impact
	"T1486": "Data Encrypted for Impact",
	"T1490": "Inhibit System Recovery",
	"T1489": "Service Stop",
	"T1529": "System Shutdown/Reboot",
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
