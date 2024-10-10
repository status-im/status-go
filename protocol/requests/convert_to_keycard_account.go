package requests

type ConvertToKeycardAccount struct {
    AccountData  string `json:"accountData"`
    SettingsJSON string `json:"settingsJSON"`
    KeycardUID   string `json:"keycardUID"`
    Password     string `json:"password"`
    NewPassword  string `json:"newPassword"`
}
