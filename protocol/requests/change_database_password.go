package requests

type ChangeDatabasePassword struct {
	KeyUID      string `json:"keyUID"`
	Password    string `json:"password"`
	NewPassword string `json:"newPassword"`
}
