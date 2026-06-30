package model

// DashboardStats holds aggregated statistics for the admin dashboard.
type DashboardStats struct {
	TotalPatients    int `json:"totalPatients"`
	TotalSessions    int `json:"totalSessions"`
	ActiveSessions   int `json:"activeSessions"`
	TodayNewPatients int `json:"todayNewPatients"`
	TodayNewSessions int `json:"todayNewSessions"`
}

// AdminPatientItem is a summary view of a patient for the admin patient list.
type AdminPatientItem struct {
	ID           string `json:"id"`
	RealName     string `json:"realName"`
	Phone        string `json:"phone"`
	Gender       string `json:"gender"`
	BirthDate    string `json:"birthDate"`
	CreatedAt    string `json:"createdAt"`
	SessionCount int    `json:"sessionCount"`
}

// AdminPatientQuery carries query parameters for listing patients.
type AdminPatientQuery struct {
	Page     int    `json:"-"`
	PageSize int    `json:"-"`
	Search   string `json:"-"`
}

// AdminPatientListResult is the paginated response for patient listing.
type AdminPatientListResult struct {
	Items    []AdminPatientItem `json:"items"`
	Total    int                `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"pageSize"`
}

// AdminSessionItem is a summary view of a visit session for the admin session list.
type AdminSessionItem struct {
	ID          string `json:"id"`
	PatientID   string `json:"patientId"`
	PatientName string `json:"patientName"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// AdminSessionQuery carries query parameters for listing sessions.
type AdminSessionQuery struct {
	Page      int    `json:"-"`
	PageSize  int    `json:"-"`
	Status    string `json:"-"`
	PatientID string `json:"-"`
}

// AdminSessionListResult is the paginated response for session listing.
type AdminSessionListResult struct {
	Items    []AdminSessionItem `json:"items"`
	Total    int                `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"pageSize"`
}
