package validator

import "github.com/go-playground/validator/v10"

func RegisterCustomValidations(validate *validator.Validate) {
	validate.RegisterValidation("lat", validateLat)
	validate.RegisterValidation("lng", validateLng)
	validate.RegisterValidation("radius_km", validateRadiusKM)
}

func validateLat(fl validator.FieldLevel) bool {
	lat := fl.Field().Float()
	return lat >= -90.0 && lat <= 90.0
}

func validateLng(fl validator.FieldLevel) bool {
	lng := fl.Field().Float()
	return lng >= -180.0 && lng <= 180.0
}

func validateRadiusKM(fl validator.FieldLevel) bool {
	radius := fl.Field().Float()
	return radius >= 0.1 && radius <= 100.0
}
