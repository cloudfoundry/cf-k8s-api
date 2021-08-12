package routes_test

import (
	"fmt"
	//"cloudfoundry.org/cf-k8s-api/routes"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"testing"
)
const routerPort = 9000
func TestRouter(t *testing.T) {
	spec.Run(t, "object", testRootV3Route, spec.Report(report.Terminal{}))
}

func testRootV3Route(t *testing.T, when spec.G, it spec.S) {

	Expect := NewWithT(t).Expect

	when("", func() {
		it("invokes the provided handler", func() {
			// initialize the router
			router := mux.NewRouter() //routes.Router{}

			var numInvoked = 0
			testFunction := func(w http.ResponseWriter, r *http.Request) {
				numInvoked++
			}


			// add handler to router
			router.HandleFunc("/v3", testFunction).Methods("GET")
			// start the router
			go func() {
				log.Fatal(http.ListenAndServe(":" + fmt.Sprint(routerPort) , router))
			}()


			// hit the router
			resp, err := http.Get("http://localhost:" +  fmt.Sprint(routerPort) + "/v3")
			if err != nil {
				log.Fatal(err)
			}

			defer resp.Body.Close()

			_, readErr := ioutil.ReadAll(resp.Body)

			if readErr != nil {
				log.Fatal(err)
			}


			// make sure handler function was called
			Expect(numInvoked).Should(Equal(1))
		})

	})
}