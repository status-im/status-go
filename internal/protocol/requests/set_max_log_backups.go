package requests

type SetMaxLogBackups struct {
	MaxLogBackups uint `json:"maxLogBackups" validate:"omitempty,gte=0"`
}
