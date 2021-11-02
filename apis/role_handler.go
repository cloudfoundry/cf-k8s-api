package apis

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"code.cloudfoundry.org/cf-k8s-api/payloads"
	"code.cloudfoundry.org/cf-k8s-api/presenter"
	"code.cloudfoundry.org/cf-k8s-api/repositories"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

const (
	RolesEndpoint = "/v3/roles"
)

//counterfeiter:generate -o fake -fake-name CFRoleRepository . CFRoleRepository

type CFRoleRepository interface {
	CreateSpaceRole(ctx context.Context, role repositories.RoleRecord) (repositories.RoleRecord, error)
}

type RoleHandler struct {
	logger     logr.Logger
	apiBaseURL url.URL
	roleRepo   CFRoleRepository
}

func NewRoleHandler(apiBaseURL url.URL, roleRepo CFRoleRepository) *RoleHandler {
	return &RoleHandler{
		logger:     controllerruntime.Log.WithName("Role Handler"),
		apiBaseURL: apiBaseURL,
		roleRepo:   roleRepo,
	}
}

func (h *RoleHandler) roleCreateHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var payload payloads.RoleCreate
	rme := DecodeAndValidatePayload(r, &payload)
	if rme != nil {
		h.logger.Error(rme, "Failed to parse body")
		writeErrorResponse(w, rme)

		return
	}

	role := payload.ToRecord()
	role.GUID = uuid.NewString()

	record, err := h.roleRepo.CreateSpaceRole(r.Context(), role)
	if err != nil {
		// if workloads.HasErrorCode(err, workloads.DuplicateSpaceRoleError) {
		// 	errorDetail := fmt.Sprintf("User %q has already the %q role assigned in space %q.", role.User, role.Type, role.Space)
		// 	h.logger.Info(errorDetail)
		// 	writeUnprocessableEntityError(w, errorDetail)
		// 	return
		// }
		h.logger.Error(err, "Failed to create role", "Role Type", role.Type, "Space", role.Space, "User", role.User)
		writeUnknownErrorResponse(w)
		return
	}

	w.WriteHeader(http.StatusCreated)
	roleResponse := presenter.ForCreateRole(record, h.apiBaseURL)
	json.NewEncoder(w).Encode(roleResponse)
}

func (h *RoleHandler) RegisterRoutes(router *mux.Router) {
	router.Path(OrgListEndpoint).Methods("POST").HandlerFunc(h.roleCreateHandler)
}
