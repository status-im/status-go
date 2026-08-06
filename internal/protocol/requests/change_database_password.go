package requests

type ChangeDatabasePassword struct {
	KeyUID      string `json:"keyUID"`
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
	Rekey       bool   `json:"rekey"` // if true, a new DEK is generated and the databases and keystore files are re-encrypted with it.
}
