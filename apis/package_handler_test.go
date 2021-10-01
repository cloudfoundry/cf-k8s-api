package apis_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/v1/remote"

	. "code.cloudfoundry.org/cf-k8s-api/apis"
	"code.cloudfoundry.org/cf-k8s-api/apis/fake"
	"code.cloudfoundry.org/cf-k8s-api/repositories"

	"github.com/gorilla/mux"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"k8s.io/client-go/rest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	testPackageHandlerLoggerName = "TestPackageHandler"
)

func TestPackage(t *testing.T) {
	spec.Run(t, "the POST /v3/packages endpoint", testPackageCreateHandler, spec.Report(report.Terminal{}))
	spec.Run(t, "the POST /v3/packages/upload endpoint", testPackageUploadHandler, spec.Report(report.Terminal{}))
}

func testPackageCreateHandler(t *testing.T, when spec.G, it spec.S) {
	g := NewWithT(t)

	var (
		rr            *httptest.ResponseRecorder
		packageRepo   *fake.CFPackageRepository
		appRepo       *fake.CFAppRepository
		clientBuilder *fake.ClientBuilder
		router        *mux.Router
	)

	makePostRequest := func(body string) {
		req, err := http.NewRequest("POST", "/v3/packages", strings.NewReader(body))
		g.Expect(err).NotTo(HaveOccurred())

		router.ServeHTTP(rr, req)
	}

	const (
		packageGUID = "the-package-guid"
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

	getRR := func() *httptest.ResponseRecorder { return rr }

	it.Before(func() {
		rr = httptest.NewRecorder()
		router = mux.NewRouter()

		packageRepo = new(fake.CFPackageRepository)
		packageRepo.CreatePackageReturns(repositories.PackageRecord{
			Type:      "bits",
			AppGUID:   appGUID,
			SpaceGUID: spaceGUID,
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

		apiHandler := NewPackageHandler(logf.Log.WithName(testPackageHandlerLoggerName), defaultServerURL, packageRepo, appRepo, clientBuilder.Spy, nil, nil, &rest.Config{}, "", "")
		apiHandler.RegisterRoutes(router)
	})

	when("on the happy path", func() {
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
			g.Expect(actualCreate).To(Equal(repositories.PackageCreateMessage{
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

		itRespondsWithUnknownError(it, g, getRR)

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
			g.Expect(rr.Code).To(Equal(http.StatusUnprocessableEntity), "Matching HTTP response code:")
		})

		it("returns Content-Type as JSON in header", func() {
			contentTypeHeader := rr.Header().Get("Content-Type")
			g.Expect(contentTypeHeader).To(Equal(jsonHeader), "Matching Content-Type header:")
		})

		it("has the expected error response body", func() {
			g.Expect(rr.Body.String()).To(MatchJSON(`{
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
			g.Expect(rr.Code).To(Equal(http.StatusUnprocessableEntity), "Matching HTTP response code:")
		})

		it("returns Content-Type as JSON in header", func() {
			contentTypeHeader := rr.Header().Get("Content-Type")
			g.Expect(contentTypeHeader).To(Equal(jsonHeader), "Matching Content-Type header:")
		})

		it("has the expected error response body", func() {
			g.Expect(rr.Body.String()).To(MatchJSON(`{
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
			g.Expect(rr.Code).To(Equal(http.StatusUnprocessableEntity), "Matching HTTP response code:")
		})

		it("returns Content-Type as JSON in header", func() {
			contentTypeHeader := rr.Header().Get("Content-Type")
			g.Expect(contentTypeHeader).To(Equal(jsonHeader), "Matching Content-Type header:")
		})

		it("has the expected error response body", func() {
			g.Expect(rr.Body.String()).To(MatchJSON(`{
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
			g.Expect(rr.Code).To(Equal(http.StatusBadRequest), "Matching HTTP response code:")
		})

		it("returns Content-Type as JSON in header", func() {
			contentTypeHeader := rr.Header().Get("Content-Type")
			g.Expect(contentTypeHeader).To(Equal(jsonHeader), "Matching Content-Type header:")
		})

		it("has the expected error response body", func() {
			g.Expect(rr.Body.String()).To(MatchJSON(`{
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

		itRespondsWithUnknownError(it, g, getRR)

		it("doesn't create a Package", func() {
			g.Expect(packageRepo.CreatePackageCallCount()).To(Equal(0))
		})
	})

	when("creating the package in the repo errors", func() {
		it.Before(func() {
			packageRepo.CreatePackageReturns(repositories.PackageRecord{}, errors.New("boom"))
			makePostRequest(validBody)
		})

		itRespondsWithUnknownError(it, g, getRR)
	})
}

func testPackageUploadHandler(t *testing.T, when spec.G, it spec.S) {
	g := NewWithT(t)
	var (
		rr                *httptest.ResponseRecorder
		packageRepo       *fake.CFPackageRepository
		appRepo           *fake.CFAppRepository
		uploadImageSource *fake.SourceImageUploader
		buildRegistryAuth *fake.RegistryAuthBuilder
		credentialOption  remote.Option
		clientBuilder     *fake.ClientBuilder
		router            *mux.Router
	)

	getRR := func() *httptest.ResponseRecorder { return rr }

	makeUploadRequest := func(packageGUID string, file io.Reader) {
		var b bytes.Buffer
		writer := multipart.NewWriter(&b)
		part, err := writer.CreateFormFile("bits", "unused.zip")
		g.Expect(err).NotTo(HaveOccurred())

		_, err = io.Copy(part, file)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(writer.Close()).To(Succeed())

		req, err := http.NewRequest("POST", fmt.Sprintf("/v3/packages/%s/upload", packageGUID), &b)
		g.Expect(err).NotTo(HaveOccurred())
		req.Header.Add("Content-Type", writer.FormDataContentType())

		router.ServeHTTP(rr, req)
	}

	const (
		packageGUID                = "the-package-guid"
		appGUID                    = "the-app-guid"
		createdAt                  = "1906-04-18T13:12:00Z"
		updatedAt                  = "1906-04-18T13:12:01Z"
		imageRefWithDigest         = "some-org/the-package-guid@SHA256:some-sha-256"
		srcFileContents            = "the-src-file-contents"
		packageRegistryBase        = "some-org"
		packageImagePullSecretName = "package-image-pull-secret"
	)

	it.Before(func() {
		rr = httptest.NewRecorder()
		router = mux.NewRouter()

		packageRepo = new(fake.CFPackageRepository)
		packageRepo.FetchPackageReturns(repositories.PackageRecord{
			Type:      "bits",
			AppGUID:   appGUID,
			SpaceGUID: spaceGUID,
			GUID:      packageGUID,
			State:     "AWAITING_UPLOAD",
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		}, nil)
		packageRepo.UpdatePackageSourceReturns(repositories.PackageRecord{
			Type:      "bits",
			AppGUID:   appGUID,
			SpaceGUID: spaceGUID,
			GUID:      packageGUID,
			State:     "PROCESSING_UPLOAD",
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		}, nil)

		uploadImageSource = new(fake.SourceImageUploader)
		uploadImageSource.Returns(imageRefWithDigest, nil)

		appRepo = new(fake.CFAppRepository)
		clientBuilder = new(fake.ClientBuilder)
		credentialOption = remote.WithUserAgent("for-test-use-only") // real one should have credentials
		buildRegistryAuth = new(fake.RegistryAuthBuilder)
		buildRegistryAuth.Returns(credentialOption, nil)

		apiHandler := NewPackageHandler(
			logf.Log.WithName(testPackageHandlerLoggerName),
			defaultServerURL,
			packageRepo,
			appRepo,
			clientBuilder.Spy,
			uploadImageSource.Spy,
			buildRegistryAuth.Spy,
			&rest.Config{},
			packageRegistryBase,
			packageImagePullSecretName,
		)

		apiHandler.RegisterRoutes(router)
	})

	when("on the happy path", func() {
		it.Before(func() {
			makeUploadRequest(packageGUID, strings.NewReader(srcFileContents))
		})

		it("returns status 200", func() {
			g.Expect(rr.Code).To(Equal(http.StatusOK), "Matching HTTP response code:")
		})

		it("returns Content-Type as JSON in header", func() {
			contentTypeHeader := rr.Header().Get("Content-Type")
			g.Expect(contentTypeHeader).To(Equal(jsonHeader), "Matching Content-Type header:")
		})

		it("configures the client", func() {
			g.Expect(clientBuilder.CallCount()).To(Equal(1))
		})

		it("fetches the right package", func() {
			g.Expect(packageRepo.FetchPackageCallCount()).To(Equal(1))

			_, _, actualPackageGUID := packageRepo.FetchPackageArgsForCall(0)
			g.Expect(actualPackageGUID).To(Equal(packageGUID))
		})

		it("uploads the image source", func() {
			g.Expect(uploadImageSource.CallCount()).To(Equal(1))
			imageRef, srcFile, actualCredentialOption := uploadImageSource.ArgsForCall(0)
			g.Expect(imageRef).To(Equal(fmt.Sprintf("%s/%s", packageRegistryBase, packageGUID)))
			actualSrcContents, err := io.ReadAll(srcFile)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(string(actualSrcContents)).To(Equal(srcFileContents))
			g.Expect(actualCredentialOption).NotTo(BeNil())
		})

		it("saves the uploaded image reference on the package", func() {
			g.Expect(packageRepo.UpdatePackageSourceCallCount()).To(Equal(1))
			_, _, message := packageRepo.UpdatePackageSourceArgsForCall(0)
			g.Expect(message.GUID).To(Equal(packageGUID))
			g.Expect(message.ImageRef).To(Equal(imageRefWithDigest))
			g.Expect(message.RegistrySecretName).To(Equal(packageImagePullSecretName))
		})

		it("returns a JSON body", func() {
			g.Expect(rr.Body.String()).To(MatchJSON(`
				{
				  "guid": "` + packageGUID + `",
				  "type": "bits",
				  "data": {},
				  "state": "PROCESSING_UPLOAD",
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

	when("the record doesn't exist", func() {
		it.Before(func() {
			packageRepo.FetchPackageReturns(repositories.PackageRecord{}, repositories.NotFoundError{})

			makeUploadRequest("no-such-package-guid", strings.NewReader("the-zip-contents"))
		})

		itRespondsWithNotFound(it, g, "Package not found", getRR)

		it("doesn't build an image from the source", func() {
			g.Expect(uploadImageSource.CallCount()).To(Equal(0))
		})

		it("doesn't update any Packages", func() {
			g.Expect(packageRepo.UpdatePackageSourceCallCount()).To(Equal(0))
		})
	})

	when("building the client errors", func() {
		it.Before(func() {
			clientBuilder.Returns(nil, errors.New("boom"))

			makeUploadRequest(packageGUID, strings.NewReader("the-zip-contents"))
		})

		itRespondsWithUnknownError(it, g, getRR)

		it("doesn't build an image from the source", func() {
			g.Expect(uploadImageSource.CallCount()).To(Equal(0))
		})

		it("doesn't update any Packages", func() {
			g.Expect(packageRepo.UpdatePackageSourceCallCount()).To(Equal(0))
		})
	})

	when("fetching the package errors", func() {
		it.Before(func() {
			packageRepo.FetchPackageReturns(repositories.PackageRecord{}, errors.New("boom"))

			makeUploadRequest(packageGUID, strings.NewReader("the-zip-contents"))
		})

		itRespondsWithUnknownError(it, g, getRR)

		it("doesn't build an image from the source", func() {
			g.Expect(uploadImageSource.CallCount()).To(Equal(0))
		})

		it("doesn't update any Packages", func() {
			g.Expect(packageRepo.UpdatePackageSourceCallCount()).To(Equal(0))
		})
	})

	when("no bits file is given", func() {
		it.Before(func() {
			var b bytes.Buffer
			writer := multipart.NewWriter(&b)
			g.Expect(writer.Close()).To(Succeed())

			req, err := http.NewRequest("POST", fmt.Sprintf("/v3/packages/%s/upload", packageGUID), &b)
			g.Expect(err).NotTo(HaveOccurred())
			req.Header.Add("Content-Type", writer.FormDataContentType())

			router.ServeHTTP(rr, req)
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
						"detail": "Upload must include bits"
					}
				]
			}`))
		})

		it("doesn't build an image from the source", func() {
			g.Expect(uploadImageSource.CallCount()).To(Equal(0))
		})

		it("doesn't update any Packages", func() {
			g.Expect(packageRepo.UpdatePackageSourceCallCount()).To(Equal(0))
		})
	})

	when("building the image credentials errors", func() {
		it.Before(func() {
			buildRegistryAuth.Returns(nil, errors.New("boom"))

			makeUploadRequest(packageGUID, strings.NewReader("the-zip-contents"))
		})

		itRespondsWithUnknownError(it, g, getRR)

		it("doesn't build an image from the source", func() {
			g.Expect(uploadImageSource.CallCount()).To(Equal(0))
		})

		it("doesn't update any Packages", func() {
			g.Expect(packageRepo.UpdatePackageSourceCallCount()).To(Equal(0))
		})
	})

	when("uploading the source image errors", func() {
		it.Before(func() {
			uploadImageSource.Returns("", errors.New("boom"))

			makeUploadRequest(packageGUID, strings.NewReader("the-zip-contents"))
		})

		itRespondsWithUnknownError(it, g, getRR)

		it("doesn't update any Packages", func() {
			g.Expect(packageRepo.UpdatePackageSourceCallCount()).To(Equal(0))
		})
	})

	when("updating the package source registry errors", func() {
		it.Before(func() {
			packageRepo.UpdatePackageSourceReturns(repositories.PackageRecord{}, errors.New("boom"))

			makeUploadRequest(packageGUID, strings.NewReader("the-zip-contents"))
		})

		itRespondsWithUnknownError(it, g, getRR)
	})
}
