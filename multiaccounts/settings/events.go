package settings

type EventSettingChanged struct {
	Setting SettingField
	Value   any
}
