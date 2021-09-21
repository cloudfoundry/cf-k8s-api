package apis_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/sclevine/spec/report"

	"code.cloudfoundry.org/cf-k8s-api/repositories"

	"k8s.io/client-go/rest"

	. "code.cloudfoundry.org/cf-k8s-api/apis"
	"code.cloudfoundry.org/cf-k8s-api/apis/fake"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
)

func TestPackage(t *testing.T) {
	spec.Run(t, "PackageCreateHandler", testPackageCreateHandler, spec.Report(report.Terminal{}))
}

func testPackageCreateHandler(t *testing.T, when spec.G, it spec.S) {
	g := NewWithT(t)

	const (
		testPackageHandlerLoggerName = "TestPackageHandler"
	)

	var (
		rr            *httptest.ResponseRecorder
		packageRepo   *fake.CFPackageRepository
		appRepo       *fake.CFAppRepository
		clientBuilder *fake.ClientBuilder
		apiHandler    *PackageHandler
	)

	makePostRequest := func(body string) {
		req, err := http.NewRequest("POST", "unused-path", strings.NewReader(body))
		g.Expect(err).NotTo(HaveOccurred())

		handler := http.HandlerFunc(apiHandler.PackageCreateHandler)
		handler.ServeHTTP(rr, req)
	}

	itRespondsWithUnknownError := func() {
		it("returns status 500 InternalServerError", func() {
			g.Expect(rr.Code).Should(Equal(http.StatusInternalServerError), "Matching HTTP response code:")
		})

		it("returns Content-Type as JSON in header", func() {
			contentTypeHeader := rr.Header().Get("Content-Type")
			g.Expect(contentTypeHeader).Should(Equal(jsonHeader), "Matching Content-Type header:")
		})

		it("returns a CF API formatted Error response", func() {
			g.Expect(rr.Body.String()).Should(MatchJSON(`{
					"errors": [
						{
							"title": "UnknownError",
							"detail": "An unknown error occurred.",
							"code": 10001
						}
					]
				}`), "Response body matches response:")
		})
	}

	const (
		packageGUID = "` + packageGUID + `"
		appGUID     = "the-app-guid"
		spaceGUID   = "the-space-guid"
		validBody   = `{
			"type": "bits",
			"relationships": {
				"app": {
					"data": {
						"guid": "` + appGUID + `"
					}
				}
        	}
		}`
		createdAt = "1906-04-18T13:12:00Z"
		updatedAt = "1906-04-18T13:12:01Z"
	)

	it.Before(func() {
		rr = httptest.NewRecorder()

		packageRepo = new(fake.CFPackageRepository)
		packageRepo.CreatePackageReturns(repositories.PackageRecord{
			Type:      "bits",
			AppGUID:   appGUID,
			GUID:      packageGUID,
			State:     "AWAITING_UPLOAD",
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		}, nil)

		appRepo = new(fake.CFAppRepository)
		appRepo.FetchAppReturns(repositories.AppRecord{
			SpaceGUID: spaceGUID,
		}, nil)

		clientBuilder = new(fake.ClientBuilder)

		apiHandler = &PackageHandler{
			ServerURL:   defaultServerURL,
			PackageRepo: packageRepo,
			AppRepo:     appRepo,
			K8sConfig:   &rest.Config{},
			Logger:      logf.Log.WithName(testPackageHandlerLoggerName),
			BuildClient: clientBuilder.Spy,
		}
	})

	when("the POST /v3/packages succeeds", func() {
		it.Before(func() {
			makePostRequest(validBody)
		})

		it("returns status 201", func() {
			g.Expect(rr.Code).To(Equal(http.StatusCreated), "Matching HTTP response code:")
		})

		it("returns Content-Type as JSON in header", func() {
			contentTypeHeader := rr.Header().Get("Content-Type")
			g.Expect(contentTypeHeader).To(Equal(jsonHeader), "Matching Content-Type header:")
		})

		it("configures the client", func() {
			g.Expect(clientBuilder.CallCount()).To(Equal(1))
		})

		it("creates a CFPackage", func() {
			g.Expect(packageRepo.CreatePackageCallCount()).To(Equal(1))
			_, _, actualCreate := packageRepo.CreatePackageArgsForCall(0)
			g.Expect(actualCreate).To(Equal(repositories.PackageCreate{
				Type:      "bits",
				AppGUID:   appGUID,
				SpaceGUID: spaceGUID,
			}))
		})

		it("returns a JSON body", func() {
			g.Expect(rr.Body.String()).To(MatchJSON(`
				{
				  "guid": "` + packageGUID + `",
				  "type": "bits",
				  "data": {},
				  "state": "AWAITING_UPLOAD",
				  "created_at": "` + createdAt + `",
				  "updated_at": "` + updatedAt + `",
				  "relationships": {
					"app": {
					  "data": {
						"guid": "` + appGUID + `"
					  }
					}
				  },
				  "links": {
					"self": {
					  "href": "` + defaultServerURI("/v3/packages/", packageGUID) + `"
					},
					"upload": {
					  "href": "` + defaultServerURI("/v3/packages/", packageGUID, "/upload") + `",
					  "method": "POST"
					},
					"download": {
					  "href": "` + defaultServerURI("/v3/packages/", packageGUID, "/download") + `",
					  "method": "GET"
					},
					"app": {
					  "href": "` + defaultServerURI("/v3/apps/", appGUID) + `"
					}
				  },
				  "metadata": {
					"labels": { },
					"annotations": { }
				  }
				}
            `))
		})
	})

	when("the app doesn't exist", func() {
		it.Before(func() {
			appRepo.FetchAppReturns(repositories.AppRecord{}, repositories.NotFoundError{})

			makePostRequest(validBody)
		})

		it("returns status 422", func() {
			g.Expect(rr.Code).To(Equal(http.StatusUnprocessableEntity), "Matching HTTP response code:")
		})

		it("returns Content-Type as JSON in header", func() {
			contentTypeHeader := rr.Header().Get("Content-Type")
			g.Expect(contentTypeHeader).To(Equal(jsonHeader), "Matching Content-Type header:")
		})

		it("responds with error code", func() {
			g.Expect(rr.Body.String()).To(MatchJSON(`{
				"errors": [
					{
						"code": 10008,
						"title": "CF-UnprocessableEntity",
						"detail": "App is invalid. Ensure it exists and you have access to it."
					}
				]
			}`))
		})

		it("doesn't create a package", func() {
			g.Expect(packageRepo.CreatePackageCallCount()).To(Equal(0))
		})
	})

	when("the app exists check returns an error", func() {
		it.Before(func() {
			appRepo.FetchAppReturns(repositories.AppRecord{}, errors.New("boom"))

			makePostRequest(validBody)
		})

		itRespondsWithUnknownError()

		it("doesn't create a package", func() {
			g.Expect(packageRepo.CreatePackageCallCount()).To(Equal(0))
		})
	})

	when("the type is invalid", func() {
		const (
			bodyWithInvalidType = `{
				"type": "docker",
				"relationships": {
					"app": {
						"data": {
							"guid": "` + appGUID + `"
						}
					}
				}
			}`
		)

		it.Before(func() {
			makePostRequest(bodyWithInvalidType)
		})

		it("returns a status 422 Unprocessable Entity", func() {
			g.Expect(rr.Code).Should(Equal(http.StatusUnprocessableEntity), "Matching HTTP response code:")
		})

		it("returns Content-Type as JSON in header", func() {
			contentTypeHeader := rr.Header().Get("Content-Type")
			g.Expect(contentTypeHeader).Should(Equal(jsonHeader), "Matching Content-Type header:")
		})

		it("has the expected error response body", func() {
			g.Expect(rr.Body.String()).Should(MatchJSON(`{
					"errors": [
						{
							"code":   10008,
							"title": "CF-UnprocessableEntity",
							"detail": "Type must be one of ['bits']"
						}
					]
				}`), "Response body matches response:")
		})
	})

	when("the relationship field is completely omitted", func() {
		it.Before(func() {
			makePostRequest(`{ "type": "bits" }`)
		})

		it("returns a status 422 Unprocessable Entity", func() {
			g.Expect(rr.Code).Should(Equal(http.StatusUnprocessableEntity), "Matching HTTP response code:")
		})

		it("returns Content-Type as JSON in header", func() {
			contentTypeHeader := rr.Header().Get("Content-Type")
			g.Expect(contentTypeHeader).Should(Equal(jsonHeader), "Matching Content-Type header:")
		})

		it("has the expected error response body", func() {
			g.Expect(rr.Body.String()).Should(MatchJSON(`{
					"errors": [
						{
							"code":   10008,
							"title": "CF-UnprocessableEntity",
							"detail": "Relationships is a required field"
						}
					]
				}`), "Response body matches response:")
		})
	})

	when("an invalid relationship is given", func() {
		const bodyWithoutAppRelationship = `{
			"type": "bits",
			"relationships": {
				"build": {
					"data": {}
			   	}
			}
		}`

		it.Before(func() {
			makePostRequest(bodyWithoutAppRelationship)
		})

		it("returns a status 422 Unprocessable Entity", func() {
			g.Expect(rr.Code).Should(Equal(http.StatusUnprocessableEntity), "Matching HTTP response code:")
		})

		it("returns Content-Type as JSON in header", func() {
			contentTypeHeader := rr.Header().Get("Content-Type")
			g.Expect(contentTypeHeader).Should(Equal(jsonHeader), "Matching Content-Type header:")
		})

		it("has the expected error response body", func() {
			g.Expect(rr.Body.String()).Should(MatchJSON(`{
					"errors": [
						{
							"code":   10008,
							"title": "CF-UnprocessableEntity",
							"detail": "invalid request body: json: unknown field \"build\""
						}
					]
				}`), "Response body matches response:")
		})
	})

	when("the JSON body is invalid", func() {
		it.Before(func() {
			makePostRequest(`{`)
		})

		it("returns a status 400 Bad Request ", func() {
			g.Expect(rr.Code).Should(Equal(http.StatusBadRequest), "Matching HTTP response code:")
		})

		it("returns Content-Type as JSON in header", func() {
			contentTypeHeader := rr.Header().Get("Content-Type")
			g.Expect(contentTypeHeader).Should(Equal(jsonHeader), "Matching Content-Type header:")
		})

		it("has the expected error response body", func() {
			g.Expect(rr.Body.String()).Should(MatchJSON(`{
					"errors": [
						{
							"title": "CF-MessageParseError",
							"detail": "Request invalid due to parse error: invalid request body",
							"code": 1001
						}
					]
				}`), "Response body matches response:")
		})
	})

	when("building the k8s client errors", func() {
		it.Before(func() {
			clientBuilder.Returns(nil, errors.New("boom"))
			makePostRequest(validBody)
		})

		it("returns a status 500 Bad Request ", func() {
			g.Expect(rr.Code).Should(Equal(http.StatusInternalServerError), "Matching HTTP response code:")
		})

		it("returns Content-Type as JSON in header", func() {
			contentTypeHeader := rr.Header().Get("Content-Type")
			g.Expect(contentTypeHeader).Should(Equal(jsonHeader), "Matching Content-Type header:")
		})

		it("has the expected error response body", func() {
			g.Expect(rr.Body.String()).Should(MatchJSON(`{
				 "errors": [
					  {
						   "title": "UnknownError",
						   "detail": "An unknown error occurred.",
						   "code": 10001
					  }
				 ]
			}`), "Response body matches response:")
		})

		it("doesn't create a Package", func() {
			g.Expect(packageRepo.CreatePackageCallCount()).To(Equal(0))
		})
	})

	when("creating the package in the repo errors", func() {
		it.Before(func() {
			packageRepo.CreatePackageReturns(repositories.PackageRecord{}, errors.New("boom"))
			makePostRequest(validBody)
		})

		it("returns a status 500 Bad Request ", func() {
			g.Expect(rr.Code).Should(Equal(http.StatusInternalServerError), "Matching HTTP response code:")
		})

		it("returns Content-Type as JSON in header", func() {
			contentTypeHeader := rr.Header().Get("Content-Type")
			g.Expect(contentTypeHeader).Should(Equal(jsonHeader), "Matching Content-Type header:")
		})

		it("has the expected error response body", func() {
			g.Expect(rr.Body.String()).Should(MatchJSON(`{
				 "errors": [
					  {
						   "title": "UnknownError",
						   "detail": "An unknown error occurred.",
						   "code": 10001
					  }
				 ]
			}`), "Response body matches response:")
		})
	})
}