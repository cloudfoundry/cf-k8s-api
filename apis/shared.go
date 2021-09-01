package apis

import (
	"fmt"

	"code.cloudfoundry.org/cf-k8s-api/presenters"
)

func newNotFoundError(resourceName string) presenters.ErrorsResponse {
	return presenters.ErrorsResponse{Errors: []presenters.PresentedError{{
		Title:  fmt.Sprintf("%s not found", resourceName),
		Detail: "CF-ResourceNotFound",
		Code:   10010,
	}}}
}

func newUnknownError() presenters.ErrorsResponse {
	return presenters.ErrorsResponse{Errors: []presenters.PresentedError{{
		Title:  "UnknownError",
		Detail: "An unknown error occurred.",
		Code:   10001,
	}}}
}

func newMessageParseError() presenters.ErrorsResponse {
	return presenters.ErrorsResponse{Errors: []presenters.PresentedError{{
		Title:  "CF-MessageParseError",
		Detail: "Request invalid due to parse error: invalid request body",
		Code:   1001,
	}}}
}

func newUnprocessableEntityError(detail string) presenters.ErrorsResponse {
	return presenters.ErrorsResponse{Errors: []presenters.PresentedError{{
		Title:  "CF-UnprocessableEntity",
		Detail: detail,
		Code:   10008,
	}}}
}
