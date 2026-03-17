package models

type FlagValueType string

const (
	FlagValueTypeBool   FlagValueType = "bool"
	FlagValueTypeString FlagValueType = "string"
	FlagValueTypeInt    FlagValueType = "int"
	FlagValueTypeFloat  FlagValueType = "float"
)

type FlagDefinition struct {
	Name    string        `yaml:"name"`
	Type    FlagValueType `yaml:"type"`
	Default any           `yaml:"default"`
}
