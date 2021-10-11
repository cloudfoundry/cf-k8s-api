package apis

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"code.cloudfoundry.org/cf-k8s-api/payloads"
	"code.cloudfoundry.org/cf-k8s-api/presenter"
	"code.cloudfoundry.org/cf-k8s-api/repositories"
	"github.com/go-http-utils/headers"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

const (
	OrgListEndpoint = "/v3/organizations"
)

//counterfeiter:generate -o fake -fake-name CFOrgRepository . CFOrgRepository

type CFOrgRepository interface {
	CreateOrg(context context.Context, org repositories.OrgRecord) (repositories.OrgRecord, error)
	FetchOrgs(context context.Context, token string, orgNames []string) ([]repositories.OrgRecord, error)
}

type OrgHandler struct {
	orgRepo    CFOrgRepository
	logger     logr.Logger
	apiBaseURL url.URL
}

func NewOrgHandler(orgRepo CFOrgRepository, apiBaseURL url.URL) *OrgHandler {
	return &OrgHandler{
		orgRepo:    orgRepo,
		apiBaseURL: apiBaseURL,
		logger:     controllerruntime.Log.WithName("Org Handler"),
	}
}

func (h *OrgHandler) orgCreateHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	var payload payloads.OrgCreate
	rme := DecodeAndValidatePayload(r, &payload)
	if rme != nil {
		writeErrorResponse(w, rme)

		return
	}

	org := payload.ToRecord()
	org.GUID = uuid.New().String()

	record, err := h.orgRepo.CreateOrg(ctx, org)
	if err != nil {
		h.logger.Error(err, "failed to create org")
		writeUnknownErrorResponse(w)

		return
	}

	w.WriteHeader(http.StatusCreated)
	orgResponse := presenter.ForCreateOrg(record, h.apiBaseURL)
	json.NewEncoder(w).Encode(orgResponse)
}

func (h *OrgHandler) orgListHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	var names []string
	namesList := r.URL.Query().Get("names")
	if len(namesList) > 0 {
		names = strings.Split(namesList, ",")
	}

	bearerToken := parseAuthorizationHeader(r.Header.Get(headers.Authorization))
	if bearerToken == "" {
		h.logger.Info("no bearer token provided")
		writeUnauthorizedErrorResponse(w)

		return
	}

	orgs, err := h.orgRepo.FetchOrgs(ctx, bearerToken, names)
	if err != nil {
		h.logger.Error(err, "failed to fetch orgs")
		writeUnknownErrorResponse(w)

		return
	}

	orgList := presenter.ForOrgList(orgs, h.apiBaseURL)
	json.NewEncoder(w).Encode(orgList)
}

func (h *OrgHandler) RegisterRoutes(router *mux.Router) {
	router.Path(OrgListEndpoint).Methods("GET").HandlerFunc(h.orgListHandler)
	router.Path(OrgListEndpoint).Methods("POST").HandlerFunc(h.orgCreateHandler)
}

func parseAuthorizationHeader(headerValue string) string {
	bearerPrefix := "Bearer "
	prefixPos := strings.LastIndex(headerValue, bearerPrefix)
	if prefixPos != 0 {
		return ""
	}
	return headerValue[len(bearerPrefix):]
}
