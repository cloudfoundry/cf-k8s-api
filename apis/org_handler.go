package apis

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"code.cloudfoundry.org/cf-k8s-api/presenter"
	"code.cloudfoundry.org/cf-k8s-api/repositories"
	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

const (
	OrgListEndpoint   = "/v3/organizations"
	SpaceListEndpoint = "/v3/spaces"
)

//counterfeiter:generate -o fake -fake-name CFOrgRepository . CFOrgRepository

type CFOrgRepository interface {
	FetchOrgs(context.Context, []string) ([]repositories.OrgRecord, error)
	FetchSpaces(context.Context, []string, []string) ([]repositories.SpaceRecord, error)
}

type OrgHandler struct {
	orgRepo    CFOrgRepository
	logger     logr.Logger
	apiBaseURL string
}

func NewOrgHandler(orgRepo CFOrgRepository, apiBaseURL string) *OrgHandler {
	return &OrgHandler{
		orgRepo:    orgRepo,
		apiBaseURL: apiBaseURL,
		logger:     controllerruntime.Log.WithName("Org Handler"),
	}
}

func (h *OrgHandler) OrgListHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	var names []string
	namesList := r.URL.Query().Get("names")
	if len(namesList) > 0 {
		names = strings.Split(namesList, ",")
	}

	orgs, err := h.orgRepo.FetchOrgs(ctx, names)
	if err != nil {
		h.logger.Error(err, "failed to fetch orgs")
		writeUnknownErrorResponse(w)

		return
	}

	orgList := presenter.ForOrgList(orgs, h.apiBaseURL)
	json.NewEncoder(w).Encode(orgList)
}

func (h *OrgHandler) SpaceListHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	orgUIDs := parseCommaSeparatedList(r.URL.Query().Get("organization_guids"))
	names := parseCommaSeparatedList(r.URL.Query().Get("names"))

	spaces, err := h.orgRepo.FetchSpaces(ctx, orgUIDs, names)
	if err != nil {
		writeUnknownErrorResponse(w)

		return
	}

	spaceList := presenter.ForSpaceList(spaces, h.apiBaseURL)
	json.NewEncoder(w).Encode(spaceList)
}

func (h *OrgHandler) RegisterRoutes(router *mux.Router) {
	router.Path(OrgListEndpoint).Methods("GET").HandlerFunc(h.OrgListHandler)
	router.Path(SpaceListEndpoint).Methods("GET").HandlerFunc(h.SpaceListHandler)
}

func parseCommaSeparatedList(list string) []string {
	var elements []string
	for _, element := range strings.Split(list, ",") {
		if element != "" {
			elements = append(elements, element)
		}
	}

	return elements
}
