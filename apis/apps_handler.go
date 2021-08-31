package apis

import (
	"code.cloudfoundry.org/cf-k8s-api/messages"
	"encoding/json"
	"net/http"
)

type AppHandler struct {
	ServerURL string
}

func (h *AppHandler) AppsCreateHandler(w http.ResponseWriter, r *http.Request) {

	//Decode request JSON into a appropriate Message Type
	var appCreateMessage messages.AppCreateMessage

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&appCreateMessage)
	if err != nil {
		w.WriteHeader(400)
		return
	}

}
