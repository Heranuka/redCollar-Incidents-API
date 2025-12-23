package validator

import (
	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
	validate.RegisterValidation("lat", func(fl validator.FieldLevel) bool {
		lat := fl.Field().Float()
		return lat >= -90 && lat <= 90
	})
	validate.RegisterValidation("lng", func(fl validator.FieldLevel) bool {
		lng := fl.Field().Float()
		return lng >= -180 && lng <= 180
	})
}

func ValidateStruct(s interface{}) error {
	return validate.Struct(s)
}
