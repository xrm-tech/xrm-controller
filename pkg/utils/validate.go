package utils

type IError struct {
	Field  string
	Reason string
	Value  string
}

type ValidateStatus struct {
	Message string
	Fields  []IError
}
