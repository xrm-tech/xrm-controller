package xrmcontroller

import "github.com/go-playground/validator/v10"

type IError struct {
	Field  string
	Reason string
	Value  string
}

type ValidateStatus struct {
	Message string
	Fields  []IError
}

type Status struct {
	Message string
}

func ValidatorError(vErr validator.ValidationErrors) ValidateStatus {
	validation := ValidateStatus{
		Message: "validation failed",
		Fields:  make([]IError, 0, len(vErr)),
	}
	for _, err := range vErr {
		el := IError{
			Field:  err.Field(),
			Reason: err.Tag(),
			Value:  err.Param(),
		}
		validation.Fields = append(validation.Fields, el)
	}
	return validation
}
