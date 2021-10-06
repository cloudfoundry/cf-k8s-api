package apis

import (
	"context"
	"encoding/json"
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"code.cloudfoundry.org/cf-k8s-api/presenter"
	"code.cloudfoundry.org/cf-k8s-api/repositories"

	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	"k8s.io/client-go/rest"
)

const (
	DropletGetEndpoint = "/v3/droplets/{guid}"
)

//counterfeiter:generate -o fake -fake-name CFDropletRepository . CFDropletRepository
type CFDropletRepository interface {
	FetchDroplet(context.Context, client.Client, string) (repositories.DropletRecord, error)
}

type DropletHandler struct {
	serverURL   string
	dropletRepo CFDropletRepository
	buildClient ClientBuilder
	logger      logr.Logger
	k8sConfig   *rest.Config
}

func NewDropletHandler(
	logger logr.Logger,
	serverURL string,
	dropletRepo CFDropletRepository,
	buildClient ClientBuilder,
	k8sConfig *rest.Config) *DropletHandler {
	return &DropletHandler{
		logger:      logger,
		serverURL:   serverURL,
		dropletRepo: dropletRepo,
		buildClient: buildClient,
		k8sConfig:   k8sConfig,
	}
}

func (h *DropletHandler) dropletGetHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	dropletGUID := vars["guid"]

	// TODO: Instantiate config based on bearer token
	// Spike code from EMEA folks around this: https://github.com/cloudfoundry/cf-crd-explorations/blob/136417fbff507eb13c92cd67e6fed6b061071941/cfshim/handlers/app_handler.go#L78
	client, err := h.buildClient(h.k8sConfig)
	if err != nil {
		h.logger.Error(err, "Unable to create Kubernetes client", "dropletGUID", dropletGUID)
		writeUnknownErrorResponse(w)
		return
	}

	droplet, err := h.dropletRepo.FetchDroplet(ctx, client, dropletGUID)
	if err != nil {
		switch err.(type) {
		case repositories.NotFoundError:
			h.logger.Info("Droplet not found", "dropletGUID", dropletGUID)
			writeNotFoundErrorResponse(w, "Droplet")
			return
		default:
			h.logger.Error(err, "Failed to fetch droplet from Kubernetes", "dropletGUID", dropletGUID)
			writeUnknownErrorResponse(w)
			return
		}
	}

	responseBody, err := json.Marshal(presenter.ForDroplet(droplet, h.serverURL))
	if err != nil {
		h.logger.Error(err, "Failed to render response", "dropletGUID", dropletGUID)
		writeUnknownErrorResponse(w)
		return
	}
	w.Write(responseBody)

}

func (h *DropletHandler) RegisterRoutes(router *mux.Router) {
	router.Path(DropletGetEndpoint).Methods("GET").HandlerFunc(h.dropletGetHandler)
}
