package apis

import (
	"fmt"
	"net/http"

	"code.cloudfoundry.org/cf-k8s-api/fetchers"
	"github.com/gorilla/mux"
	"k8s.io/client-go/rest"
)

type AppsHandler struct {
	ServerURL   string
	AppsFetcher fetchers.AppsFetcher
	K8sConfig   *rest.Config // TODO: this would be global for all requests, not what we want
}

func (h *AppsHandler) AppsGetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	appGUID := vars["guid"]

	// TODO: Instantiate config based on bearer token
	// Spike code from EMEA folks around this: https://github.com/cloudfoundry/cf-crd-explorations/blob/136417fbff507eb13c92cd67e6fed6b061071941/cfshim/handlers/app_handler.go#L78
	err := h.AppsFetcher.ConfigureClient(h.K8sConfig)
	if err != nil {
		w.WriteHeader(500)
	}

	app, err := h.AppsFetcher.FetchApp(appGUID)
	if err != nil {
		w.WriteHeader(500)
	}
	w.Write([]byte(fmt.Sprintf("{\"guid\": \"%s\"}", app.GUID)))
}
