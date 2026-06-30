package model

// SystemSettings holds the configurable system settings.
type SystemSettings struct {
	SiteName              string `json:"siteName"`
	MaxConcurrentSessions int    `json:"maxConcurrentSessions"`
	SessionTimeoutMinutes int    `json:"sessionTimeoutMinutes"`
	EnableRegistration    bool   `json:"enableRegistration"`
}

// UpdateSystemSettingsInput carries the fields for partial settings update.
// Nil/zero-value fields are not applied.
type UpdateSystemSettingsInput struct {
	SiteName              *string `json:"siteName"`
	MaxConcurrentSessions *int    `json:"maxConcurrentSessions"`
	SessionTimeoutMinutes *int    `json:"sessionTimeoutMinutes"`
	EnableRegistration    *bool   `json:"enableRegistration"`
}
