package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"code.cloudfoundry.org/cf-k8s-api/apis"
	"code.cloudfoundry.org/cf-k8s-api/routes"
	"github.com/gorilla/mux"

	. "code.cloudfoundry.org/cf-k8s-api/config"
)

const defaultConfigPath = "config.json"

// Inserting a change to test the GH Action PR workflow
func main() {
	configPath := os.Getenv("CONFIG")
	if configPath == "" {
		configPath = defaultConfigPath
	}
	fmt.Printf("Config path: %s", configPath)

	config, err := LoadConfigFromPath(configPath)
	if err != nil {
		errorMessage := fmt.Sprintf("Config could not be read: %v", err)
		panic(errorMessage)
	}

	// Configure the RootV3 API Handler
	apiRootV3Handler := &apis.RootV3Handler{
		ServerURL: config.ServerURL,
	}
	apiRootHandler := &apis.RootHandler{
		ServerURL: config.ServerURL,
	}

	router := mux.NewRouter()
	// create API routes
	apiRoutes := routes.APIRoutes{
		//add API routes to handler
		RootV3Handler: apiRootV3Handler.RootV3GetHandler,
		RootHandler:   apiRootHandler.RootGetHandler,
	}
	// Call RegisterRoutes to register all the routes in APIRoutes
	apiRoutes.RegisterRoutes(router)

	portString := fmt.Sprintf(":%v", config.ServerPort)
	log.Fatal(http.ListenAndServe(portString, router))

}
