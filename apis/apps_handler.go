package apis

import (
	"code.cloudfoundry.org/cf-k8s-api/messages"
	"encoding/json"
	"errors"
	"net/http"

	"code.cloudfoundry.org/cf-k8s-api/presenters"
	"code.cloudfoundry.org/cf-k8s-api/repositories"
	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . CFAppRepository
type CFAppRepository interface {
	ConfigureClient(*rest.Config) (client.Client, error)
	FetchApp(client.Client, string) (repositories.AppRecord, error)
	FetchNamespace(client.Client, string) (repositories.SpaceRecord, error)
}

type AppHandler struct {
	ServerURL string
	AppRepo   CFAppRepository
	Logger    logr.Logger
	K8sConfig *rest.Config // TODO: this would be global for all requests, not what we want
}

func (h *AppHandler) AppsGetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	appGUID := vars["guid"]

	// TODO: Instantiate config based on bearer token
	// Spike code from EMEA folks around this: https://github.com/cloudfoundry/cf-crd-explorations/blob/136417fbff507eb13c92cd67e6fed6b061071941/cfshim/handlers/app_handler.go#L78
	client, err := h.AppRepo.ConfigureClient(h.K8sConfig)
	if err != nil {
		h.Logger.Error(err, "Unable to create Kubernetes client", "AppGUID", appGUID)
		writeUnknownErrorResponse(w)
		return
	}

	app, err := h.AppRepo.FetchApp(client, appGUID)
	if err != nil {
		switch err.(type) {
		case repositories.NotFoundError:
			h.Logger.Info("App not found", "AppGUID", appGUID)
			writeNotFoundErrorResponse(w)
			return
		default:
			h.Logger.Error(err, "Failed to fetch app from Kubernetes", "AppGUID", appGUID)
			writeUnknownErrorResponse(w)
			return
		}
	}

	responseBody, err := json.Marshal(presenters.NewPresentedApp(app, h.ServerURL))
	if err != nil {
		h.Logger.Error(err, "Failed to render response", "AppGUID", appGUID)
		writeUnknownErrorResponse(w)
		return
	}

	w.Write(responseBody)
}

func (h *AppHandler) AppsCreateHandler(w http.ResponseWriter, r *http.Request) {

	var appCreateMessage messages.AppCreateMessage
	err := DecodePayload(r, &appCreateMessage)
	if err != nil {
		var rme *requestMalformedError
		if errors.As(err, &rme) {
			writeErrorResponse(w, rme)
		} else {
			h.Logger.Error(err, "Unknown internal server error")
			writeUnknownErrorResponse(w)
		}
		return
	}

	// TODO: Instantiate config based on bearer token
	// Spike code from EMEA folks around this: https://github.com/cloudfoundry/cf-crd-explorations/blob/136417fbff507eb13c92cd67e6fed6b061071941/cfshim/handlers/app_handler.go#L78
	client, err := h.AppRepo.ConfigureClient(h.K8sConfig)
	if err != nil {
		h.Logger.Error(err, "Unable to create Kubernetes client")
		writeUnknownErrorResponse(w)
		return
	}

	namespaceGUID := appCreateMessage.Relationships.Space.Data.GUID
	_, err = h.AppRepo.FetchNamespace(client, namespaceGUID)
	if err != nil {
		switch err.(type) {
		case repositories.PermissionDeniedOrNotFoundError:
			h.Logger.Info("Namespace not found", "Namespace GUID", namespaceGUID)
			writeUnprocessableEntityError(w, "Invalid space. Ensure that the space exists and you have access to it.")
			return
		default:
			h.Logger.Error(err, "Failed to fetch namespace from Kubernetes", "Namespace GUID", namespaceGUID)
			writeUnknownErrorResponse(w)
			return
		}
	}
}

func writeErrorResponse(w http.ResponseWriter, rme *requestMalformedError) {
	w.WriteHeader(rme.httpStatus)
	responseBody, err := json.Marshal(rme.errorResponse)
	if err != nil {
		w.WriteHeader(rme.httpStatus)
		return
	}
	w.Write(responseBody)
}

func writeNotFoundErrorResponse(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	responseBody, err := json.Marshal(newNotFoundError("App"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(responseBody)
}

func writeUnknownErrorResponse(w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	responseBody, err := json.Marshal(newUnknownError())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(responseBody)
}

func writeMessageParseError(w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
	responseBody, err := json.Marshal(newMessageParseError())
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Write(responseBody)
}

func writeUnprocessableEntityError(w http.ResponseWriter, errorDetail string) {
	w.WriteHeader(http.StatusUnprocessableEntity)
	responseBody, err := json.Marshal(newUnprocessableEntityError(errorDetail))
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	w.Write(responseBody)
}
