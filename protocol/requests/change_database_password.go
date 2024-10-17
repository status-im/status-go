package requests

type ChangeDatabasePassword struct {
	KeyUID      string `json:"keyUID"`
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}
