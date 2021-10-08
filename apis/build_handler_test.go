package apis_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"code.cloudfoundry.org/cf-k8s-api/repositories"

	. "code.cloudfoundry.org/cf-k8s-api/apis"
	"code.cloudfoundry.org/cf-k8s-api/apis/fake"

	"github.com/gorilla/mux"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/rest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	testBuildHandlerLoggerName = "TestBuildHandler"
)

var _ = Describe("BuildHandler", func() {
	Describe("the GET /v3/builds/{guid} endpoint", func() {
		const (
			appGUID     = "test-app-guid"
			packageGUID = "test-package-guid"
			buildGUID   = "test-build-guid"

			stagingMem  = 1024
			stagingDisk = 2048

			createdAt = "1906-04-18T13:12:00Z"
			updatedAt = "1906-04-18T13:12:01Z"
		)

		var (
			rr            *httptest.ResponseRecorder
			req           *http.Request
			buildRepo     *fake.CFBuildRepository
			clientBuilder *fake.ClientBuilder
			router        *mux.Router
		)

		getRR := func() *httptest.ResponseRecorder { return rr }

		// set up happy path defaults
		BeforeEach(func() {
			buildRepo = new(fake.CFBuildRepository)
			buildRepo.FetchBuildReturns(repositories.BuildRecord{
				GUID:            buildGUID,
				State:           "STAGING",
				CreatedAt:       createdAt,
				UpdatedAt:       updatedAt,
				StagingMemoryMB: stagingMem,
				StagingDiskMB:   stagingDisk,
				Lifecycle: repositories.Lifecycle{
					Type: "buildpack",
					Data: repositories.LifecycleData{
						Buildpacks: []string{},
						Stack:      "",
					},
				},
				PackageGUID: packageGUID,
				AppGUID:     appGUID,
			}, nil)

			var err error
			req, err = http.NewRequest("GET", "/v3/builds/"+buildGUID, nil)
			Expect(err).NotTo(HaveOccurred())

			rr = httptest.NewRecorder()
			router = mux.NewRouter()
			clientBuilder = new(fake.ClientBuilder)

			serverURL, err := url.Parse(defaultServerURL)
			Expect(err).NotTo(HaveOccurred())
			buildHandler := NewBuildHandler(
				logf.Log.WithName(testBuildHandlerLoggerName),
				*serverURL,
				buildRepo,
				new(fake.CFPackageRepository),
				clientBuilder.Spy,
				&rest.Config{},
			)
			buildHandler.RegisterRoutes(router)
		})

		When("on the happy path", func() {
			When("build staging is not complete", func() {
				BeforeEach(func() {
					router.ServeHTTP(rr, req)
				})

				It("returns status 200 OK", func() {
					Expect(rr.Code).To(Equal(http.StatusOK), "Matching HTTP response code:")
				})

				It("returns Content-Type as JSON in header", func() {
					contentTypeHeader := rr.Header().Get("Content-Type")
					Expect(contentTypeHeader).To(Equal(jsonHeader), "Matching Content-Type header:")
				})

				It("returns the Build in the response", func() {
					Expect(rr.Body.String()).To(MatchJSON(`{
					"guid": "`+buildGUID+`",
					"created_at": "`+createdAt+`",
					"updated_at": "`+updatedAt+`",
					"created_by": {},
					"state": "STAGING",
					"staging_memory_in_mb": `+fmt.Sprint(stagingMem)+`,
					"staging_disk_in_mb": `+fmt.Sprint(stagingDisk)+`,
					"error": null,
					"lifecycle": {
						"type": "buildpack",
						"data": {
							"buildpacks": [],
							"stack": ""
						}
					},
					"package": {
						"guid": "`+packageGUID+`"
					},
					"droplet": null,
					"relationships": {
						"app": {
							"data": {
								"guid": "`+appGUID+`"
							}
						}
					},
					"metadata": {
						"labels": {},
						"annotations": {}
					},
					"links": {
						"self": {
							"href": "`+defaultServerURI("/v3/builds/", buildGUID)+`"
						},
						"app": {
							"href": "`+defaultServerURI("/v3/apps/", appGUID)+`"
						}
					}
				}`), "Response body matches response:")
				})
			})
			When("build staging is successful", func() {
				BeforeEach(func() {
					buildRepo.FetchBuildReturns(repositories.BuildRecord{
						GUID:            buildGUID,
						State:           "STAGED",
						CreatedAt:       createdAt,
						UpdatedAt:       updatedAt,
						StagingMemoryMB: stagingMem,
						StagingDiskMB:   stagingDisk,
						Lifecycle: repositories.Lifecycle{
							Type: "buildpack",
							Data: repositories.LifecycleData{
								Buildpacks: []string{},
								Stack:      "",
							},
						},
						PackageGUID: packageGUID,
						DropletGUID: buildGUID,
						AppGUID:     appGUID,
					}, nil)
					router.ServeHTTP(rr, req)
				})

				It("returns status 200 OK", func() {
					Expect(rr.Code).To(Equal(http.StatusOK), "Matching HTTP response code:")
				})

				It("returns Content-Type as JSON in header", func() {
					contentTypeHeader := rr.Header().Get("Content-Type")
					Expect(contentTypeHeader).To(Equal(jsonHeader), "Matching Content-Type header:")
				})

				It("returns the Build in the response", func() {
					Expect(rr.Body.String()).To(MatchJSON(`{
					"guid": "`+buildGUID+`",
					"created_at": "`+createdAt+`",
					"updated_at": "`+updatedAt+`",
					"created_by": {},
					"state": "STAGED",
					"staging_memory_in_mb": `+fmt.Sprint(stagingMem)+`,
					"staging_disk_in_mb": `+fmt.Sprint(stagingDisk)+`,
					"error": null,
					"lifecycle": {
						"type": "buildpack",
						"data": {
							"buildpacks": [],
							"stack": ""
						}
					},
					"package": {
						"guid": "`+packageGUID+`"
					},
					"droplet": {
						"guid": "`+buildGUID+`"
					},
					"relationships": {
						"app": {
							"data": {
								"guid": "`+appGUID+`"
							}
						}
					},
					"metadata": {
						"labels": {},
						"annotations": {}
					},
					"links": {
						"self": {
							"href": "`+defaultServerURI("/v3/builds/", buildGUID)+`"
						},
						"app": {
							"href": "`+defaultServerURI("/v3/apps/", appGUID)+`"
						},
						"droplet": {
							"href": "`+defaultServerURI("/v3/droplets/", buildGUID)+`"
						}
					}
				}`), "Response body matches response:")
				})
			})
			When("build staging fails", func() {
				const (
					stagingErrorMsg = "StagingError: something went wrong during staging"
				)
				BeforeEach(func() {
					buildRepo.FetchBuildReturns(repositories.BuildRecord{
						GUID:            buildGUID,
						State:           "FAILED",
						CreatedAt:       createdAt,
						UpdatedAt:       updatedAt,
						StagingErrorMsg: stagingErrorMsg,
						StagingMemoryMB: stagingMem,
						StagingDiskMB:   stagingDisk,
						Lifecycle: repositories.Lifecycle{
							Type: "buildpack",
							Data: repositories.LifecycleData{
								Buildpacks: []string{},
								Stack:      "",
							},
						},
						PackageGUID: packageGUID,
						DropletGUID: "",
						AppGUID:     appGUID,
					}, nil)
					router.ServeHTTP(rr, req)
				})

				It("returns status 200 OK", func() {
					Expect(rr.Code).To(Equal(http.StatusOK), "Matching HTTP response code:")
				})

				It("returns Content-Type as JSON in header", func() {
					contentTypeHeader := rr.Header().Get("Content-Type")
					Expect(contentTypeHeader).To(Equal(jsonHeader), "Matching Content-Type header:")
				})

				It("returns the Build in the response", func() {
					Expect(rr.Body.String()).To(MatchJSON(`{
					"guid": "`+buildGUID+`",
					"created_at": "`+createdAt+`",
					"updated_at": "`+updatedAt+`",
					"created_by": {},
					"state": "FAILED",
					"staging_memory_in_mb": `+fmt.Sprint(stagingMem)+`,
					"staging_disk_in_mb": `+fmt.Sprint(stagingDisk)+`,
					"error": "`+stagingErrorMsg+`",
					"lifecycle": {
						"type": "buildpack",
						"data": {
							"buildpacks": [],
							"stack": ""
						}
					},
					"package": {
						"guid": "`+packageGUID+`"
					},
					"droplet": null,
					"relationships": {
						"app": {
							"data": {
								"guid": "`+appGUID+`"
							}
						}
					},
					"metadata": {
						"labels": {},
						"annotations": {}
					},
					"links": {
						"self": {
							"href": "`+defaultServerURI("/v3/builds/", buildGUID)+`"
						},
						"app": {
							"href": "`+defaultServerURI("/v3/apps/", appGUID)+`"
						}
					}
				}`), "Make sure there is no droplet and error is surfaced from record")
				})
			})
		})

		When("building the k8s client errors", func() {
			BeforeEach(func() {
				clientBuilder.Returns(nil, errors.New("boom"))
				router.ServeHTTP(rr, req)
			})

			itRespondsWithUnknownError(getRR)
		})

		When("the build cannot be found", func() {
			BeforeEach(func() {
				buildRepo.FetchBuildReturns(repositories.BuildRecord{}, repositories.NotFoundError{})

				router.ServeHTTP(rr, req)
			})

			itRespondsWithNotFound("Build not found", getRR)
		})

		When("there is some other error fetching the build", func() {
			BeforeEach(func() {
				buildRepo.FetchBuildReturns(repositories.BuildRecord{}, errors.New("unknown!"))

				router.ServeHTTP(rr, req)
			})

			itRespondsWithUnknownError(getRR)
		})
	})
	Describe("the POST /v3/builds endpoint", func() {
		var (
			rr            *httptest.ResponseRecorder
			packageRepo   *fake.CFPackageRepository
			buildRepo     *fake.CFBuildRepository
			clientBuilder *fake.ClientBuilder
			router        *mux.Router
		)

		makePostRequest := func(body string) {
			req, err := http.NewRequest("POST", "/v3/builds", strings.NewReader(body))
			Expect(err).NotTo(HaveOccurred())

			router.ServeHTTP(rr, req)
		}

		const (
			packageGUID = "the-package-guid"
			appGUID     = "the-app-guid"
			buildGUID   = "test-build-guid"

			expectedStagingMem     = 1024
			expectedStagingDisk    = 1024
			expectedLifecycleType  = "buildpack"
			expectedLifecycleStack = "cflinuxfs3"
			spaceGUID              = "the-space-guid"
			validBody              = `{
			"package": {
				"guid": "` + packageGUID + `"
        	}
		}`
			createdAt = "1906-04-18T13:12:00Z"
			updatedAt = "1906-04-18T13:12:01Z"
		)

		getRR := func() *httptest.ResponseRecorder { return rr }

		BeforeEach(func() {
			rr = httptest.NewRecorder()
			router = mux.NewRouter()

			packageRepo = new(fake.CFPackageRepository)
			packageRepo.FetchPackageReturns(repositories.PackageRecord{
				Type:      "bits",
				AppGUID:   appGUID,
				SpaceGUID: spaceGUID,
				GUID:      packageGUID,
				State:     "READY",
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			}, nil)

			buildRepo = new(fake.CFBuildRepository)
			buildRepo.CreateBuildReturns(repositories.BuildRecord{
				GUID:            buildGUID,
				State:           "STAGING",
				CreatedAt:       createdAt,
				UpdatedAt:       updatedAt,
				StagingMemoryMB: expectedStagingMem,
				StagingDiskMB:   expectedStagingDisk,
				Lifecycle: repositories.Lifecycle{
					Type: expectedLifecycleType,
					Data: repositories.LifecycleData{
						Buildpacks: []string{},
						Stack:      expectedLifecycleStack,
					},
				},
				PackageGUID: packageGUID,
				AppGUID:     appGUID,
			}, nil)

			serverURL, err := url.Parse(defaultServerURL)
			Expect(err).NotTo(HaveOccurred())
			clientBuilder = new(fake.ClientBuilder)
			buildHandler := NewBuildHandler(
				logf.Log.WithName(testBuildHandlerLoggerName),
				*serverURL,
				buildRepo,
				packageRepo,
				clientBuilder.Spy,
				&rest.Config{},
			)
			buildHandler.RegisterRoutes(router)
		})

		When("on the happy path", func() {
			BeforeEach(func() {
				makePostRequest(validBody)
			})

			It("returns status 201", func() {
				Expect(rr.Code).To(Equal(http.StatusCreated), "Matching HTTP response code:")
			})

			It("returns Content-Type as JSON in header", func() {
				contentTypeHeader := rr.Header().Get("Content-Type")
				Expect(contentTypeHeader).To(Equal(jsonHeader), "Matching Content-Type header:")
			})

			It("configures the client", func() {
				Expect(clientBuilder.CallCount()).To(Equal(1))
			})

			When("examining the BuildCreate message", func() {
				var actualCreate repositories.BuildCreateMessage
				BeforeEach(func() {
					Expect(buildRepo.CreateBuildCallCount()).To(Equal(1), "buildRepo CreateBuild was not called")
					_, _, actualCreate = buildRepo.CreateBuildArgsForCall(0)
				})
				It("has the same SpaceGUID as the package", func() {
					Expect(actualCreate.SpaceGUID).To(Equal(spaceGUID))
				})
				It("has the same AppGUID as the package", func() {
					Expect(actualCreate.AppGUID).To(Equal(appGUID))
				})
				It("has the same PackageGUID as the request", func() {
					Expect(actualCreate.PackageGUID).To(Equal(packageGUID))
				})
				It("fills in values for StagingMemoryMB", func() {
					Expect(actualCreate.StagingMemoryMB).To(Equal(expectedStagingMem))
				})
				It("fills in values for StagingDiskMB", func() {
					Expect(actualCreate.StagingDiskMB).To(Equal(expectedStagingDisk))
				})
				It("fills in values for Lifecycle", func() {
					Expect(actualCreate.Lifecycle.Type).To(Equal(expectedLifecycleType))
					Expect(actualCreate.Lifecycle.Data.Buildpacks).To(Equal([]string{}))
					Expect(actualCreate.Lifecycle.Data.Stack).To(Equal(expectedLifecycleStack))
				})
			})

			It("returns the Build in the response", func() {
				Expect(rr.Body.String()).To(MatchJSON(`{
					"guid": "`+buildGUID+`",
					"created_at": "`+createdAt+`",
					"updated_at": "`+updatedAt+`",
					"created_by": {},
					"state": "STAGING",
					"staging_memory_in_mb": `+fmt.Sprint(expectedStagingMem)+`,
					"staging_disk_in_mb": `+fmt.Sprint(expectedStagingDisk)+`,
					"error": null,
					"lifecycle": {
						"type": "`+expectedLifecycleType+`",
						"data": {
							"buildpacks": [],
							"stack": "`+expectedLifecycleStack+`"
						}
					},
					"package": {
						"guid": "`+packageGUID+`"
					},
					"droplet": null,
					"relationships": {
						"app": {
							"data": {
								"guid": "`+appGUID+`"
							}
						}
					},
					"metadata": {
						"labels": {},
						"annotations": {}
					},
					"links": {
						"self": {
							"href": "`+defaultServerURI("/v3/builds/", buildGUID)+`"
						},
						"app": {
							"href": "`+defaultServerURI("/v3/apps/", appGUID)+`"
						}
					}
				}`), "Response body matches response:")
			})
		})

		itDoesntCreateABuild := func() {
			It("doesn't create a build", func() {
				Expect(buildRepo.CreateBuildCallCount()).To(Equal(0))
			})
		}

		When("the package doesn't exist", func() {
			BeforeEach(func() {
				packageRepo.FetchPackageReturns(repositories.PackageRecord{}, repositories.NotFoundError{})
				makePostRequest(validBody)
			})

			itRespondsWithUnprocessableEntity("Unable to use package. Ensure that the package exists and you have access to it.", getRR)
			itDoesntCreateABuild()
		})

		When("the package exists check returns an error", func() {
			BeforeEach(func() {
				packageRepo.FetchPackageReturns(repositories.PackageRecord{}, errors.New("boom"))

				makePostRequest(validBody)
			})

			itRespondsWithUnknownError(getRR)
			itDoesntCreateABuild()
		})

		When("building the k8s client errors", func() {
			BeforeEach(func() {
				clientBuilder.Returns(nil, errors.New("boom"))
				makePostRequest(validBody)
			})

			itRespondsWithUnknownError(getRR)
			itDoesntCreateABuild()
		})

		When("creating the build in the repo errors", func() {
			BeforeEach(func() {
				buildRepo.CreateBuildReturns(repositories.BuildRecord{}, errors.New("boom"))
				makePostRequest(validBody)
			})

			itRespondsWithUnknownError(getRR)
		})

		When("the JSON body is invalid", func() {
			BeforeEach(func() {
				makePostRequest(`{`)
			})

			It("returns a status 400 Bad Request ", func() {
				Expect(rr.Code).To(Equal(http.StatusBadRequest), "Matching HTTP response code:")
			})

			It("returns Content-Type as JSON in header", func() {
				contentTypeHeader := rr.Header().Get("Content-Type")
				Expect(contentTypeHeader).To(Equal(jsonHeader), "Matching Content-Type header:")
			})

			It("has the expected error response body", func() {
				Expect(rr.Body.String()).To(MatchJSON(`{
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
	})
})
