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
	"code.cloudfoundry.org/cf-k8s-api/presenters"
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
			expectedBody, err := json.Marshal(presenters.AppResponse{
				Name:  "test-app",
				GUID:  "test-app-guid",
				State: "STOPPED",
				Relationships: presenters.Relationships{
					"space": presenters.Relationship{
						GUID: "test-space-guid",
					},
				},
				Lifecycle: presenters.Lifecycle{Data: presenters.LifecycleData{
					Buildpacks: []string{},
					Stack:      "",
				}},
				Metadata: presenters.Metadata{
					Labels:      nil,
					Annotations: nil,
				},
				Links: presenters.AppLinks{
					Self: presenters.Link{
						HREF: defaultServerURI("/v3/apps/test-app-guid"),
					},
					Space: presenters.Link{
						HREF: defaultServerURI("/v3/spaces/test-space-guid"),
					},
					Processes: presenters.Link{
						HREF: defaultServerURI("/v3/apps/test-app-guid/processes"),
					},
					Packages: presenters.Link{
						HREF: defaultServerURI("/v3/apps/test-app-guid/packages"),
					},
					EnvironmentVariables: presenters.Link{
						HREF: defaultServerURI("/v3/apps/test-app-guid/environment_variables"),
					},
					CurrentDroplet: presenters.Link{
						HREF: defaultServerURI("/v3/apps/test-app-guid/droplets/current"),
					},
					Droplets: presenters.Link{
						HREF: defaultServerURI("/v3/apps/test-app-guid/droplets"),
					},
					Tasks: presenters.Link{},
					StartAction: presenters.Link{
						HREF:   defaultServerURI("/v3/apps/test-app-guid/actions/start"),
						Method: "POST",
					},
					StopAction: presenters.Link{
						HREF:   defaultServerURI("/v3/apps/test-app-guid/actions/stop"),
						Method: "POST",
					},
					Revisions:         presenters.Link{},
					DeployedRevisions: presenters.Link{},
					Features:          presenters.Link{},
				},
			})

			g.Expect(err).NotTo(HaveOccurred())
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
			expectedBody, err := json.Marshal(presenters.ErrorsResponse{Errors: []presenters.PresentedError{{
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
			expectedBody, err := json.Marshal(presenters.ErrorsResponse{Errors: []presenters.PresentedError{{
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
