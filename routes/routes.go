package routes

import (
	"github.com/gorilla/mux"
	"net/http"
)

	"github.com/gorilla/mux"
)

// Just contains the CF API Routes and maps them to handler functions

const (
	RootGetEndpoint   = "/"
	RootV3GetEndpoint = "/v3"
	AppsGetEndpoint   = "/v3/apps/{guid}"
	AppsCreateEndpoint = RootV3GetEndpoint + "/apps"
)

type httpHandlerFunction func(w http.ResponseWriter, r *http.Request)

type APIRoutes struct {
	RootV3Handler httpHandlerFunction
	RootHandler   httpHandlerFunction
	AppsHandler   httpHandlerFunction
	AppsCreateHandler httpHandlerFunction
}

func (a *APIRoutes) RegisterRoutes(router *mux.Router) {
	if a.RootV3Handler == nil || a.RootHandler == nil || a.AppsHandler == nil || a.AppsCreateHandler == nil{
		panic("APIRoutes: handler was nil")
	}
	router.HandleFunc(RootGetEndpoint, a.RootHandler).Methods("GET")
	router.HandleFunc(RootV3GetEndpoint, a.RootV3Handler).Methods("GET")
	router.HandleFunc(AppsCreateEndpoint, a.AppsCreateHandler).Methods("POST")
	router.HandleFunc(AppsGetEndpoint, a.AppsHandler).Methods("GET")
}