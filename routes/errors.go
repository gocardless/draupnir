package routes

import (
	"encoding/json"
	"net/http"
)

type APIError struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Code   string `json:"code"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Source struct {
		Pointer   string `json:"pointer"`
		Parameter string `json:"parameter"`
	} `json:"source"`
}

func RenderError(w http.ResponseWriter, statuscode int, err APIError) {
	w.WriteHeader(statuscode)
	json.NewEncoder(w).Encode(err)
}

var internalServerError = APIError{
	ID:     "internal_server_error",
	Code:   "internal_server_error",
	Status: "500",
	Title:  "Internal Server Error",
	Detail: "Something went wrong :(",
}

var notFoundError = APIError{
	ID:     "resource_not_found",
	Code:   "resource_not_found",
	Status: "404",
	Title:  "Resource Not Found",
	Detail: "The resource your requested could not be found",
}

// func RenderInvalidParameterError(w http.ResponseWriter, param string) {
// 	w.WriteHeader(400)
// 	json.NewEncoder(w).Encode(APIError{
// 		ID:     "invalid_parameter",
// 		Code:   "invalid_parameter",
// 		Status: "400",
// 		Title:  "Invalid Parameter",
// 		Detail: "One of your parameters is invalid",
// 		Source: {
// 			Parameter: param,
// 		},
// 	})
// }
