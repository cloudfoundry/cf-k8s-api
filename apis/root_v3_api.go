package apis

import (
	"net/http"
)

type RootV3Handler struct {
	ServerURL string
}

func (h *RootV3Handler) RootV3GetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"links":{"self":{"href":"` + h.ServerURL + `/v3"}}}`))
}
