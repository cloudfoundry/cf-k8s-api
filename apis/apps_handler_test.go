package apis_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	workloadsv1alpha1 "code.cloudfoundry.org/cf-k8s-controllers/api/v1alpha1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"code.cloudfoundry.org/cf-k8s-api/apis"
	"code.cloudfoundry.org/cf-k8s-api/presenters"
	"code.cloudfoundry.org/cf-k8s-api/repositories"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type FakeAppRepo struct {
	FetchAppFunc       func(_ client.Client, _ string) (repositories.AppRecord, error)
	FetchNamespaceFunc func(_ client.Client, _ string) (repositories.SpaceRecord, error)
}

var (
	FetchAppResponseApp    repositories.AppRecord
	FetchNamespaceResponse repositories.SpaceRecord
	FetchAppErr            error
	FetchNamespaceErr      error
)

func (f *FakeAppRepo) ConfigureClient(config *rest.Config) (client.Client, error) {
	err := workloadsv1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, err
	}

	fakeClientBuilder := &fake.ClientBuilder{}
	return fakeClientBuilder.WithScheme(scheme.Scheme).WithObjects(&workloadsv1alpha1.CFApp{}).Build(), nil
}

func (f *FakeAppRepo) FetchApp(client client.Client, appGUID string) (repositories.AppRecord, error) {
	return f.FetchAppFunc(client, appGUID)
}

func (f *FakeAppRepo) FetchNamespace(client client.Client, nsGUID string) (repositories.SpaceRecord, error) {
	return f.FetchNamespaceFunc(client, nsGUID)
}

func TestApps(t *testing.T) {
	spec.Run(t, "AppsGetHandler", testAppsGetHandler, spec.Report(report.Terminal{}))
	spec.Run(t, "AppsCreateHandler", testAppsCreateHandler, spec.Report(report.Terminal{}))
}

func testAppsGetHandler(t *testing.T, when spec.G, it spec.S) {
	Expect := NewWithT(t).Expect

	var (
		rr *httptest.ResponseRecorder
	)

	when("the GET /v3/apps/:guid  endpoint returns successfully", func() {
		it.Before(func() {
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
			FetchAppErr = nil

			req, err := http.NewRequest("GET", "/v3/apps/my-app-guid", nil)
			Expect(err).NotTo(HaveOccurred())

			rr = httptest.NewRecorder()
			apiHandler := apis.AppHandler{
				ServerURL: defaultServerURL,
				AppRepo: &FakeAppRepo{
					FetchAppFunc: func(_ client.Client, _ string) (repositories.AppRecord, error) {
						return FetchAppResponseApp, FetchAppErr
					},
				},
				Logger:    logf.Log.WithName("TestAppHandler"),
				K8sConfig: &rest.Config{},
			}

			handler := http.HandlerFunc(apiHandler.AppsGetHandler)

			handler.ServeHTTP(rr, req)
		})

		it("returns status 200 OK", func() {
			httpStatus := rr.Code
			Expect(httpStatus).Should(Equal(http.StatusOK), "Matching HTTP response code:")
		})

		it("returns Content-Type as JSON in header", func() {
			contentTypeHeader := rr.Header().Get("Content-Type")
			Expect(contentTypeHeader).Should(Equal(jsonHeader), "Matching Content-Type header:")
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
						HREF: "https://api.example.org/v3/apps/test-app-guid",
					},
					Space: presenters.Link{
						HREF: "https://api.example.org/v3/spaces/test-space-guid",
					},
					Processes: presenters.Link{
						HREF: "https://api.example.org/v3/apps/test-app-guid/processes",
					},
					Packages: presenters.Link{
						HREF: "https://api.example.org/v3/apps/test-app-guid/packages",
					},
					EnvironmentVariables: presenters.Link{
						HREF: "https://api.example.org/v3/apps/test-app-guid/environment_variables",
					},
					CurrentDroplet: presenters.Link{
						HREF: "https://api.example.org/v3/apps/test-app-guid/droplets/current",
					},
					Droplets: presenters.Link{
						HREF: "https://api.example.org/v3/apps/test-app-guid/droplets",
					},
					Tasks: presenters.Link{},
					StartAction: presenters.Link{
						HREF:   "https://api.example.org/v3/apps/test-app-guid/actions/start",
						Method: "POST",
					},
					StopAction: presenters.Link{
						HREF:   "https://api.example.org/v3/apps/test-app-guid/actions/stop",
						Method: "POST",
					},
					Revisions:         presenters.Link{},
					DeployedRevisions: presenters.Link{},
					Features:          presenters.Link{},
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(rr.Body.String()).Should(MatchJSON(expectedBody), "Response body matches response:")
		})
	})

	when("the app cannot be found", func() {
		it.Before(func() {
			FetchAppResponseApp = repositories.AppRecord{}
			FetchAppErr = repositories.NotFoundError{Err: errors.New("not found")}

			req, err := http.NewRequest("GET", "/v3/apps/my-app-guid", nil)
			Expect(err).NotTo(HaveOccurred())

			rr = httptest.NewRecorder()
			apiHandler := apis.AppHandler{
				ServerURL: defaultServerURL,
				AppRepo: &FakeAppRepo{
					FetchAppFunc: func(_ client.Client, _ string) (repositories.AppRecord, error) {
						return FetchAppResponseApp, FetchAppErr
					},
				},
				Logger:    logf.Log.WithName("TestAppHandler"),
				K8sConfig: &rest.Config{},
			}

			handler := http.HandlerFunc(apiHandler.AppsGetHandler)

			handler.ServeHTTP(rr, req)
		})

		it("returns a CF API formatted Error response", func() {
			expectedBody, err := json.Marshal(presenters.ErrorsResponse{Errors: []presenters.PresentedError{{
				Title:  "App not found",
				Detail: "CF-ResourceNotFound",
				Code:   10010,
			}}})

			httpStatus := rr.Code
			Expect(httpStatus).Should(Equal(http.StatusNotFound), "Matching HTTP response code:")

			Expect(err).NotTo(HaveOccurred())
			Expect(rr.Body.String()).Should(MatchJSON(expectedBody), "Response body matches response:")
		})
	})

	when("there is some other error fetching the app", func() {
		it.Before(func() {
			FetchAppResponseApp = repositories.AppRecord{}
			FetchAppErr = errors.New("unknown!")

			req, err := http.NewRequest("GET", "/v3/apps/my-app-guid", nil)
			Expect(err).NotTo(HaveOccurred())

			rr = httptest.NewRecorder()
			apiHandler := apis.AppHandler{
				ServerURL: defaultServerURL,
				AppRepo: &FakeAppRepo{
					FetchAppFunc: func(_ client.Client, _ string) (repositories.AppRecord, error) {
						return FetchAppResponseApp, FetchAppErr
					},
				},
				Logger:    logf.Log.WithName("TestAppHandler"),
				K8sConfig: &rest.Config{},
			}

			handler := http.HandlerFunc(apiHandler.AppsGetHandler)

			handler.ServeHTTP(rr, req)
		})

		it("returns a CF API formatted Error response", func() {
			expectedBody, err := json.Marshal(presenters.ErrorsResponse{Errors: []presenters.PresentedError{{
				Title:  "UnknownError",
				Detail: "An unknown error occurred.",
				Code:   10001,
			}}})

			httpStatus := rr.Code
			Expect(httpStatus).Should(Equal(http.StatusInternalServerError), "Matching HTTP response code:")

			Expect(err).NotTo(HaveOccurred())
			Expect(rr.Body.String()).Should(MatchJSON(expectedBody), "Response body matches response:")
		})
	})

}

func testAppsCreateHandler(t *testing.T, when spec.G, it spec.S) {
	Expect := NewWithT(t).Expect
	var rr *httptest.ResponseRecorder

	const (
		jsonHeader       = "application/json"
		defaultServerURL = "https://api.example.org"
	)

	when("the POST /v3/apps endpoint is invoked and", func() {

		when("the request body is invalid", func() {
			it.Before(func() {
				requestBody := []byte(`{"description" : "Invalid Request"}`)

				req, err := http.NewRequest("POST", "/v3/apps", bytes.NewReader(requestBody))
				Expect(err).NotTo(HaveOccurred())

				apiHandler := apis.AppHandler{
					ServerURL: defaultServerURL,
					AppRepo: &FakeAppRepo{
						FetchAppFunc: func(_ client.Client, _ string) (repositories.AppRecord, error) {
							return FetchAppResponseApp, FetchAppErr
						},
					},
					Logger:    logf.Log.WithName("TestAppHandler"),
					K8sConfig: &rest.Config{},
				}

				rr = httptest.NewRecorder()
				handler := http.HandlerFunc(apiHandler.AppsCreateHandler)
				handler.ServeHTTP(rr, req)

			})
			it("returns a status 400 Bad Request ", func() {
				Expect(rr.Code).To(Equal(http.StatusBadRequest))
			})
			it("has the expected error response body", func() {
				expectedBody, err := json.Marshal(presenters.ErrorsResponse{Errors: []presenters.PresentedError{{
					Title:  "CF-MessageParseError",
					Detail: "Request invalid due to parse error: invalid request body",
					Code:   1001,
				}}})
				Expect(err).NotTo(HaveOccurred())
				Expect(rr.Body).To(MatchJSON(expectedBody))
			})

		})

		when("the request body is invalid with invalid app name", func() {
			it.Before(func() {
				requestBody := []byte(`{
										"name": 12345,
										"relationships": {
										  "space": {
											"data": {
											  "guid": "2f35885d-0c9d-4423-83ad-fd05066f8576"
											}
										  }
										}
									  }`)

				req, err := http.NewRequest("POST", "/v3/apps", bytes.NewReader(requestBody))
				Expect(err).NotTo(HaveOccurred())

				apiHandler := apis.AppHandler{
					ServerURL: defaultServerURL,
					AppRepo: &FakeAppRepo{
						FetchAppFunc: func(_ client.Client, _ string) (repositories.AppRecord, error) {
							return FetchAppResponseApp, FetchAppErr
						},
					},
					Logger:    logf.Log.WithName("TestAppHandler"),
					K8sConfig: &rest.Config{},
				}

				rr = httptest.NewRecorder()
				handler := http.HandlerFunc(apiHandler.AppsCreateHandler)
				handler.ServeHTTP(rr, req)

			})
			it("returns a status 422 Unprocessable Entity", func() {
				Expect(rr.Code).To(Equal(http.StatusUnprocessableEntity))
			})
			it("has the expected error response body", func() {
				expectedBody, err := json.Marshal(presenters.ErrorsResponse{Errors: []presenters.PresentedError{{
					Title:  "CF-UnprocessableEntity",
					Detail: "Name must be a string",
					Code:   10008,
				}}})
				Expect(err).NotTo(HaveOccurred())
				Expect(rr.Body).To(MatchJSON(expectedBody))
			})

		})

		when("the request body is invalid with invalid environment variable object", func() {
			it.Before(func() {
				requestBody := []byte(`{
										"name": "my_app",
										"environment_variables": [],
										"relationships": {
										  "space": {
											"data": {
											  "guid": "2f35885d-0c9d-4423-83ad-fd05066f8576"
											}
										  }
										}
									  }`)

				req, err := http.NewRequest("POST", "/v3/apps", bytes.NewReader(requestBody))
				Expect(err).NotTo(HaveOccurred())

				apiHandler := apis.AppHandler{
					ServerURL: defaultServerURL,
					AppRepo: &FakeAppRepo{
						FetchAppFunc: func(_ client.Client, _ string) (repositories.AppRecord, error) {
							return FetchAppResponseApp, FetchAppErr
						},
					},
					Logger:    logf.Log.WithName("TestAppHandler"),
					K8sConfig: &rest.Config{},
				}

				rr = httptest.NewRecorder()
				handler := http.HandlerFunc(apiHandler.AppsCreateHandler)
				handler.ServeHTTP(rr, req)

			})
			it("returns a status 422 Unprocessable Entity", func() {
				Expect(rr.Code).To(Equal(http.StatusUnprocessableEntity))
			})
			it("has the expected error response body", func() {
				expectedBody, err := json.Marshal(presenters.ErrorsResponse{Errors: []presenters.PresentedError{{
					Title:  "CF-UnprocessableEntity",
					Detail: "Environment_variables must be a map[string]string",
					Code:   10008,
				}}})
				Expect(err).NotTo(HaveOccurred())
				Expect(rr.Body).To(MatchJSON(expectedBody))
			})

		})

		when("the request body is invalid with missing required name field", func() {
			it.Before(func() {
				requestBody := []byte(`{
										"relationships": {
										  "space": {
											"data": {
											  "guid": "0c78dd5d-c723-4f2e-b168-df3c3e1d0806"
											}
										  }
										}
									  }`)

				req, err := http.NewRequest("POST", "/v3/apps", bytes.NewReader(requestBody))
				Expect(err).NotTo(HaveOccurred())

				apiHandler := apis.AppHandler{
					ServerURL: defaultServerURL,
					AppRepo: &FakeAppRepo{
						FetchAppFunc: func(_ client.Client, _ string) (repositories.AppRecord, error) {
							return FetchAppResponseApp, FetchAppErr
						},
					},
					Logger:    logf.Log.WithName("TestAppHandler"),
					K8sConfig: &rest.Config{},
				}

				rr = httptest.NewRecorder()
				handler := http.HandlerFunc(apiHandler.AppsCreateHandler)
				handler.ServeHTTP(rr, req)

			})
			it("returns a status 422 Unprocessable Entity", func() {
				Expect(rr.Code).To(Equal(http.StatusUnprocessableEntity))
			})
			it("has the expected error response body", func() {
				expectedBody, err := json.Marshal(presenters.ErrorsResponse{Errors: []presenters.PresentedError{{
					Title:  "CF-UnprocessableEntity",
					Detail: "Name must be a string",
					Code:   10008,
				}}})
				Expect(err).NotTo(HaveOccurred())
				Expect(rr.Body).To(MatchJSON(expectedBody))
			})

		})

		when("the request body is invalid with missing data within lifecycle", func() {
			it.Before(func() {
				requestBody := []byte(`{
										"name": "test-app",
										"lifecycle":{},
										"relationships": {
										  "space": {
											"data": {
											  "guid": "0c78dd5d-c723-4f2e-b168-df3c3e1d0806"
											}
										  }
										}
									  }`)

				req, err := http.NewRequest("POST", "/v3/apps", bytes.NewReader(requestBody))
				Expect(err).NotTo(HaveOccurred())

				apiHandler := apis.AppHandler{
					ServerURL: defaultServerURL,
					AppRepo: &FakeAppRepo{
						FetchAppFunc: func(_ client.Client, _ string) (repositories.AppRecord, error) {
							return FetchAppResponseApp, FetchAppErr
						},
					},
					Logger:    logf.Log.WithName("TestAppHandler"),
					K8sConfig: &rest.Config{},
				}

				rr = httptest.NewRecorder()
				handler := http.HandlerFunc(apiHandler.AppsCreateHandler)
				handler.ServeHTTP(rr, req)

			})
			it("returns a status 422 Unprocessable Entity", func() {
				Expect(rr.Code).To(Equal(http.StatusUnprocessableEntity))
			})
			it("has the expected error response body", func() {
				expectedBody, err := json.Marshal(presenters.ErrorsResponse{Errors: []presenters.PresentedError{{
					Title:  "CF-UnprocessableEntity",
					Detail: "Buildpacks must be a []string,Stack must be a string",
					Code:   10008,
				}}})
				Expect(err).NotTo(HaveOccurred())
				Expect(rr.Body).To(MatchJSON(expectedBody))
			})

		})

		when("the space does not exists", func() {
			it.Before(func() {
				FetchNamespaceResponse = repositories.SpaceRecord{}
				FetchNamespaceErr = repositories.PermissionDeniedOrNotFoundError{Err: errors.New("not found")}
				requestBody := []byte(`{
										"name": "test-app",
										"relationships": {
										  "space": {
											"data": {
											  "guid": "0c78dd5d-c723-4f2e-b168-df3c3e1d0806"
											}
										  }
										}
									  }`)

				req, err := http.NewRequest("POST", "/v3/apps", bytes.NewReader(requestBody))
				Expect(err).NotTo(HaveOccurred())

				rr = httptest.NewRecorder()
				apiHandler := apis.AppHandler{
					ServerURL: defaultServerURL,
					AppRepo: &FakeAppRepo{
						FetchNamespaceFunc: func(_ client.Client, _ string) (repositories.SpaceRecord, error) {
							return FetchNamespaceResponse, FetchNamespaceErr
						},
					},
					Logger:    logf.Log.WithName("TestAppHandler"),
					K8sConfig: &rest.Config{},
				}

				handler := http.HandlerFunc(apiHandler.AppsCreateHandler)

				handler.ServeHTTP(rr, req)
			})

			it("returns a CF API formatted Error response", func() {
				expectedBody, err := json.Marshal(presenters.ErrorsResponse{Errors: []presenters.PresentedError{{
					Detail: "Invalid space. Ensure that the space exists and you have access to it.",
					Title:  "CF-UnprocessableEntity",
					Code:   10008,
				}}})

				httpStatus := rr.Code
				Expect(httpStatus).Should(Equal(http.StatusUnprocessableEntity), "Matching HTTP response code:")

				Expect(err).NotTo(HaveOccurred())
				Expect(rr.Body.String()).Should(MatchJSON(expectedBody), "Response body matches response:")
			})
		})

	})
}
