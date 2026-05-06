package inertia

import (
	"encoding/json"
	"net/http"
)

// PrecognitionSuccess writes a successful Precognition validation response.
func PrecognitionSuccess(w http.ResponseWriter) {
	w.Header().Set(HeaderPrecognition, "true")
	w.Header().Set(HeaderPrecognitionSuccess, "true")
	AppendVary(w.Header(), HeaderPrecognition)
	w.WriteHeader(http.StatusNoContent)
}

// PrecognitionErrors writes a failed Precognition validation response.
func PrecognitionErrors(w http.ResponseWriter, errors ValidationErrors) error {
	if errors == nil {
		errors = ValidationErrors{}
	}
	w.Header().Set(HeaderPrecognition, "true")
	AppendVary(w.Header(), HeaderPrecognition)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	return json.NewEncoder(w).Encode(Props{"errors": Props(errors)})
}
