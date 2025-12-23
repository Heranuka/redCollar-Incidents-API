package middleware

import (
	"encoding/json"
	"net/http"
	"redCollar/pkg/validator"
)

func BindJSON(target interface{}) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if err := json.NewDecoder(r.Body).Decode(target); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}

			if err := validator.ValidateStruct(target); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			next.ServeHTTP(w, r.WithContext(r.Context()))
		}
	}
}
