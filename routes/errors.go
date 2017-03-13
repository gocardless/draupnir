package routes

import (
	"encoding/json"
	"net/http"
)

type APIError struct {
	ID     string      `json:"id"`
	Status string      `json:"status"`
	Code   string      `json:"code"`
	Title  string      `json:"title"`
	Detail string      `json:"detail"`
	Source ErrorSource `json:"source,omitempty"`
}

type ErrorSource struct {
	Pointer   string `json:"pointer,omitempty"`
	Parameter string `json:"parameter,omitempty"`
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
	Detail: "The resource you requested could not be found",
}

var imageNotFoundError = APIError{
	ID:     "resource_not_found",
	Code:   "resource_not_found",
	Status: "404",
	Title:  "Image Not Found",
	Detail: "The image you specified could not be found",
}

var badImageIDError = APIError{
	ID:     "bad_request",
	Code:   "bad_request",
	Status: "400",
	Title:  "Bad Request",
	Detail: "The image ID provided is not valid",
	Source: ErrorSource{
		Parameter: "image_id",
	},
}

var unreadyImageError = APIError{
	ID:     "unprocessable_entity",
	Code:   "unprocessable_entity",
	Status: "422",
	Title:  "Image Not Ready",
	Detail: "The specified image is not ready to be used",
	Source: ErrorSource{
		Parameter: "image_id",
	},
}

var cannotDeleteImageWithInstancesError = APIError{
	ID:     "unprocessable_entity",
	Code:   "unprocessable_entity",
	Status: "422",
	Title:  "Image Not Ready",
	Detail: "Cannot delete an image that has instances",
}

var invalidJSONError = APIError{
	ID:     "bad_request",
	Code:   "bad_request",
	Status: "400",
	Title:  "Invalid JSON",
	Detail: "Your JSON is malformed",
}
