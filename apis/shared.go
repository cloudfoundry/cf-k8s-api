package apis

import (
	"encoding/json"
	"fmt"
	"net/http"

	"code.cloudfoundry.org/cf-k8s-api/presenter"
)

func newNotFoundError(resourceName string) presenter.ErrorsResponse {
	return presenter.ErrorsResponse{Errors: []presenter.PresentedError{{
		Title:  fmt.Sprintf("%s not found", resourceName),
		Detail: "CF-ResourceNotFound",
		Code:   10010,
	}}}
}

func newUnknownError() presenter.ErrorsResponse {
	return presenter.ErrorsResponse{Errors: []presenter.PresentedError{{
		Title:  "UnknownError",
		Detail: "An unknown error occurred.",
		Code:   10001,
	}}}
}

func writeNotFoundErrorResponse(w http.ResponseWriter, resourceName string) {
	w.WriteHeader(http.StatusNotFound)
	responseBody, err := json.Marshal(newNotFoundError(resourceName))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(responseBody)
}

func writeUnknownErrorResponse(w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	responseBody, err := json.Marshal(newUnknownError())
	if err != nil {
		return
	}
	_, _ = w.Write(responseBody)
}
