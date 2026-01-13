package services

type FlagValueType string

const (
	FlagValueTypeBool   FlagValueType = "bool"
	FlagValueTypeString FlagValueType = "string"
)

type FlagDefinition struct {
	Name    string
	Type    FlagValueType
	Default any
}

var FrontendFlagDefinitions = []FlagDefinition{
	{Name: "dark_mode", Type: FlagValueTypeBool, Default: false},
	{Name: "maintenance_mode", Type: FlagValueTypeBool, Default: false},
	{Name: "feedback_enabled", Type: FlagValueTypeBool, Default: true},
	{Name: "force_update_enabled", Type: FlagValueTypeBool, Default: false},
	{Name: "minimum_app_version", Type: FlagValueTypeString, Default: "1.0.0"},
}
