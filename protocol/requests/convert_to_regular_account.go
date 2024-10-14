package requests

type ConvertToRegularAccount struct {
	Mnemonic     string `json:"mnemonic"`
	CurrPassword string `json:"currPassword"`
	NewPassword  string `json:"newPassword"`
}
