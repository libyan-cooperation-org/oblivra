package osquery

// QueryTemplate represents a pre-built osquery SQL template
type QueryTemplate struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	SQL         string `json:"sql"`
	Category    string `json:"category"`
}

// GetDefaultQueries returns the built-in osquery library
func GetDefaultQueries() []QueryTemplate {
	return []QueryTemplate{
		// Security
		{
			Name:        "Fileless Malware Check",
			Description: "Processes running without a binary on disk",
			Category:    "Security",
			SQL:         "SELECT pid, name, path, cmdline FROM processes WHERE on_disk = 0;",
		},
		{
			Name:        "Listening Ports",
			Description: "All open listening ports and associated processes",
			Category:    "Security",
			SQL:         "SELECT lp.pid, lp.port, lp.protocol, lp.address, p.name FROM listening_ports lp JOIN processes p ON lp.pid = p.pid;",
		},
		{
			Name:        "Last Logins",
			Description: "Recent user logins",
			Category:    "Security",
			SQL:         "SELECT username, time, host, type FROM last ORDER BY time DESC LIMIT 20;",
		},
		{
			Name:        "Authorized SSH Keys",
			Description: "All authorized_keys entries",
			Category:    "Security",
			SQL:         "SELECT uid, algorithm, key, comment FROM authorized_keys;",
		},
		{
			Name:        "SUID Binaries",
			Description: "Files with SUID/SGID bits set (potential privilege escalation)",
			Category:    "Security",
			SQL:         "SELECT path, username, permissions FROM suid_bin;",
		},
		{
			Name:        "Crontab Entries",
			Description: "Scheduled cron jobs across all users",
			Category:    "Security",
			SQL:         "SELECT event, command, path FROM crontab;",
		},

		// Performance
		{
			Name:        "High CPU Processes",
			Description: "Top 10 processes by CPU usage",
			Category:    "Performance",
			SQL:         "SELECT pid, name, uid, resident_size, total_size, user_time, system_time FROM processes ORDER BY user_time + system_time DESC LIMIT 10;",
		},
		{
			Name:        "Memory Hogs",
			Description: "Top 10 processes by memory consumption",
			Category:    "Performance",
			SQL:         "SELECT pid, name, resident_size, total_size FROM processes ORDER BY resident_size DESC LIMIT 10;",
		},
		{
			Name:        "Disk Usage",
			Description: "Mounted filesystem usage",
			Category:    "Performance",
			SQL:         "SELECT device, path, type, blocks_size, blocks_free, blocks FROM mounts;",
		},
		{
			Name:        "System Uptime",
			Description: "How long the system has been running",
			Category:    "Performance",
			SQL:         "SELECT days, hours, minutes, total_seconds FROM uptime;",
		},

		// Inventory
		{
			Name:        "System Info",
			Description: "Hostname, OS, architecture",
			Category:    "Inventory",
			SQL:         "SELECT hostname, cpu_brand, physical_memory, hardware_vendor, hardware_model FROM system_info;",
		},
		{
			Name:        "OS Version",
			Description: "Operating system details",
			Category:    "Inventory",
			SQL:         "SELECT name, version, major, minor, build, platform FROM os_version;",
		},
		{
			Name:        "Users",
			Description: "Local user accounts",
			Category:    "Inventory",
			SQL:         "SELECT uid, gid, username, description, directory, shell FROM users;",
		},
		{
			Name:        "Docker Containers",
			Description: "Running Docker containers",
			Category:    "Inventory",
			SQL:         "SELECT id, name, image, status, state FROM docker_containers;",
		},
		{
			Name:        "Network Interfaces",
			Description: "Network adapter configuration",
			Category:    "Inventory",
			SQL:         "SELECT interface, address, mask, type FROM interface_addresses;",
		},
	}
}
