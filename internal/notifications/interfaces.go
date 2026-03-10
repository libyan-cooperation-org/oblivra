package notifications

// Notifier defines the interface for multi-channel alert dispatch.
type Notifier interface {
	SendAlert(title, message string)
	UpdateConfig(cfg NotificationConfig)
	GetConfig() NotificationConfig
}
