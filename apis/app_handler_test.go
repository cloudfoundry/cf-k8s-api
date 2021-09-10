package apis_test

import (
	"code.cloudfoundry.org/cf-k8s-api/apis/apisfakes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"code.cloudfoundry.org/cf-k8s-api/apis"
	"code.cloudfoundry.org/cf-k8s-api/presenter"
	"code.cloudfoundry.org/cf-k8s-api/repositories"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"k8s.io/client-go/rest"
)

var (
	FetchAppResponseApp repositories.AppRecord
	FetchAppErr         error
)

func TestApp(t *testing.T) {
	spec.Run(t, "object", testAppGetHandler, spec.Report(report.Terminal{}))
}

func testAppGetHandler(t *testing.T, when spec.G, it spec.S) {
	g := NewWithT(t)

	var (
		rr *httptest.ResponseRecorder
	)

	when("the GET /v3/apps/:guid endpoint returns successfully", func() {
		it.Before(func() {
			fakeAppRepo := &apisfakes.FakeCFAppRepository{}
			FetchAppResponseApp = repositories.AppRecord{
				GUID:      "test-app-guid",
				Name:      "test-app",
				SpaceGUID: "test-space-guid",
				State:     repositories.DesiredState("STOPPED"),
				Lifecycle: repositories.Lifecycle{
					Data: repositories.LifecycleData{
						Buildpacks: []string{},
						Stack:      "",
					},
				},
			}
			fakeAppRepo.FetchAppReturns(FetchAppResponseApp, nil)

			req, err := http.NewRequest("GET", "/v3/apps/my-app-guid", nil)
			g.Expect(err).NotTo(HaveOccurred())

			rr = httptest.NewRecorder()
			apiHandler := apis.AppHandler{
				ServerURL: defaultServerURL,
				AppRepo:   fakeAppRepo,
				Logger:    logf.Log.WithName("TestAppHandler"),
				K8sConfig: &rest.Config{},
			}

			handler := http.HandlerFunc(apiHandler.AppGetHandler)

			handler.ServeHTTP(rr, req)
		})

		it("returns status 200 OK", func() {
			httpStatus := rr.Code
			g.Expect(httpStatus).Should(Equal(http.StatusOK), "Matching HTTP response code:")
		})

		it("returns Content-Type as JSON in header", func() {
			contentTypeHeader := rr.Header().Get("Content-Type")
			g.Expect(contentTypeHeader).Should(Equal(jsonHeader), "Matching Content-Type header:")
		})

		it("returns the App in the response", func() {
			expectedBody := `{
				"name": "test-app",
				"guid": "test-app-guid",
				"state": "STOPPED",
				"relationships": {
					"space": {
						"guid": "test-space-guid"
					}
				},
				"lifecycle": {
					"data": {
						"buildpacks": [],
						"stack": ""
					}
				},
				"metadata": {
					"labels": {},
					"annotations": {}
				},
				"links": {
					"self": {
						"href": "https://api.example.org/v3/apps/test-app-guid"
					},
					"space": {
						"href": "https://api.example.org/v3/spaces/test-space-guid"
					},
					"processes": {
						"href": "https://api.example.org/v3/apps/test-app-guid/processes"
					},
					"packages": {
						"href": "https://api.example.org/v3/apps/test-app-guid/packages"
					},
					"environment_variables": {
						"href": "https://api.example.org/v3/apps/test-app-guid/environment_variables"
					},
					"current_droplet": {
						"href": "https://api.example.org/v3/apps/test-app-guid/droplets/current"
					},
					"droplets": {
						"href": "https://api.example.org/v3/apps/test-app-guid/droplets"
					},
					"tasks": {},
					"start": {
						"href": "https://api.example.org/v3/apps/test-app-guid/actions/start",
                  		"method": "POST"
					},
					"stop": {
						"href": "https://api.example.org/v3/apps/test-app-guid/actions/stop",
						"method": "POST"
					},
					"revisions": {},
					"deployed_revisions": {},
					"features": {}
				}
			}`

			g.Expect(rr.Body.String()).Should(MatchJSON(expectedBody), "Response body matches response:")
		})
	})

	when("the app cannot be found", func() {
		it.Before(func() {
			fakeAppRepo := &apisfakes.FakeCFAppRepository{}
			fakeAppRepo.FetchAppReturns(repositories.AppRecord{}, repositories.NotFoundError{Err: errors.New("not found")})

			req, err := http.NewRequest("GET", "/v3/apps/my-app-guid", nil)
			g.Expect(err).NotTo(HaveOccurred())

			rr = httptest.NewRecorder()
			apiHandler := apis.AppHandler{
				ServerURL: defaultServerURL,
				AppRepo:   fakeAppRepo,
				Logger:    logf.Log.WithName("TestAppHandler"),
				K8sConfig: &rest.Config{},
			}

			handler := http.HandlerFunc(apiHandler.AppGetHandler)

			handler.ServeHTTP(rr, req)
		})

		it("returns a CF API formatted Error response", func() {
			expectedBody, err := json.Marshal(presenter.ErrorsResponse{Errors: []presenter.PresentedError{{
				Title:  "App not found",
				Detail: "CF-ResourceNotFound",
				Code:   10010,
			}}})

			httpStatus := rr.Code
			g.Expect(httpStatus).Should(Equal(http.StatusNotFound), "Matching HTTP response code:")

			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(rr.Body.String()).Should(MatchJSON(expectedBody), "Response body matches response:")
		})
	})

	when("there is some other error fetching the app", func() {
		it.Before(func() {
			fakeAppRepo := &apisfakes.FakeCFAppRepository{}
			fakeAppRepo.FetchAppReturns(repositories.AppRecord{}, errors.New("unknown"))

			req, err := http.NewRequest("GET", "/v3/apps/my-app-guid", nil)
			g.Expect(err).NotTo(HaveOccurred())

			rr = httptest.NewRecorder()
			apiHandler := apis.AppHandler{
				ServerURL: defaultServerURL,
				AppRepo:   fakeAppRepo,
				Logger:    logf.Log.WithName("TestAppHandler"),
				K8sConfig: &rest.Config{},
			}

			handler := http.HandlerFunc(apiHandler.AppGetHandler)

			handler.ServeHTTP(rr, req)
		})

		it("returns a CF API formatted Error response", func() {
			expectedBody, err := json.Marshal(presenter.ErrorsResponse{Errors: []presenter.PresentedError{{
				Title:  "UnknownError",
				Detail: "An unknown error occurred.",
				Code:   10001,
			}}})

			httpStatus := rr.Code
			g.Expect(httpStatus).Should(Equal(http.StatusInternalServerError), "Matching HTTP response code:")

			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(rr.Body.String()).Should(MatchJSON(expectedBody), "Response body matches response:")
		})
	})

}
