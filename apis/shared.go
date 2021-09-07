package apis

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-playground/validator/v10"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"strings"

	"code.cloudfoundry.org/cf-k8s-api/presenters"
)

var Logger = ctrl.Log.WithName("Shared Handler Functions")

type requestMalformedError struct {
	httpStatus    int
	errorResponse presenters.ErrorsResponse
}

func (rme *requestMalformedError) Error() string {
	return fmt.Sprintf("Error throwing an http %v", rme.httpStatus)
}

func DecodePayload(r *http.Request, object interface{}) error {

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&object)
	if err != nil {
		var unmarshalTypeError *json.UnmarshalTypeError
		switch {
		case errors.As(err, &unmarshalTypeError):
			Logger.Error(err, fmt.Sprintf("Request body contains an invalid value for the %q field (should be of type %v)", strings.Title(unmarshalTypeError.Field), unmarshalTypeError.Type))
			return &requestMalformedError{
				httpStatus:    http.StatusUnprocessableEntity,
				errorResponse: newUnprocessableEntityError(fmt.Sprintf("%v must be a %v", strings.Title(unmarshalTypeError.Field), unmarshalTypeError.Type)),
			}
		default:
			Logger.Error(err, "Unable to parse the JSON body")
			return &requestMalformedError{
				httpStatus:    http.StatusBadRequest,
				errorResponse: newMessageParseError(),
			}
		}
	}

	v := validator.New()
	err = v.Struct(object)

	if err != nil {
		var errorMessages []string
		for _, e := range err.(validator.ValidationErrors) {
			errorMessages = append(errorMessages, fmt.Sprintf("%v must be a %v", strings.Title(e.Field()), e.Type()))
		}

		if len(errorMessages) > 0 {
			return &requestMalformedError{
				httpStatus:    http.StatusUnprocessableEntity,
				errorResponse: newUnprocessableEntityError(strings.Join(errorMessages[:], ",")),
			}
		}
	}

	return nil
}

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
