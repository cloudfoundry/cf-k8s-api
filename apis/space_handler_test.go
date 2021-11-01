package apis_test

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"code.cloudfoundry.org/cf-k8s-api/apis"
	"code.cloudfoundry.org/cf-k8s-api/apis/fake"
	"code.cloudfoundry.org/cf-k8s-api/repositories"
	"code.cloudfoundry.org/cf-k8s-controllers/webhooks/workloads"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Spaces", func() {
	const spacesBase = "/v3/spaces"

	var (
		now           time.Time
		spaceHandler  *apis.SpaceHandler
		spaceRepo     *fake.CFSpaceRepository
		requestMethod string
		requestBody   string
		requestPath   string
	)

	BeforeEach(func() {
		now = time.Unix(1631892190, 0) // 2021-09-17T15:23:10Z
		requestBody = ""
		requestPath = spacesBase
		spaceRepo = new(fake.CFSpaceRepository)
		spaceHandler = apis.NewSpaceHandler(spaceRepo, *serverURL)
		spaceHandler.RegisterRoutes(router)
	})

	JustBeforeEach(func() {
		req, err := http.NewRequestWithContext(ctx, requestMethod, requestPath, strings.NewReader(requestBody))
		Expect(err).NotTo(HaveOccurred())

		router.ServeHTTP(rr, req)
	})

	Describe("Create Space", func() {
		BeforeEach(func() {
			requestMethod = http.MethodPost

			spaceRepo.CreateSpaceReturns(repositories.SpaceRecord{
				Name:             "the-space",
				GUID:             "t-h-e-s-p-a-c-e",
				OrganizationGUID: "the-org",
				CreatedAt:        now,
				UpdatedAt:        now,
			}, nil)

			requestBody = `{
                "name": "the-space",
                "relationships": {
                    "organization": {
                        "data": {
                            "guid": "[org-guid]"
                        }
                    }
                }
            }`
		})

		It("returns 201 with appropriate success JSON", func() {
			Expect(rr).To(HaveHTTPStatus(http.StatusCreated))
			Expect(rr).To(HaveHTTPHeaderWithValue("Content-Type", "application/json"))
			Expect(rr).To(HaveHTTPBody(MatchJSON(fmt.Sprintf(`{
                "guid": "t-h-e-s-p-a-c-e",
                "name": "the-space",
                "created_at": "2021-09-17T15:23:10Z",
                "updated_at": "2021-09-17T15:23:10Z",
                "metadata": {
                    "labels": {},
                    "annotations": {}
                },
                "relationships": {
                    "organization": {
                        "data": {
                            "guid": "the-org"
                        }
                    }
                },
                "links": {
                    "self": {
                        "href": "%[1]s/v3/spaces/t-h-e-s-p-a-c-e"
                    },
                    "organization": {
                        "href": "%[1]s/v3/organizations/the-org"
                    }
                }
            }`, defaultServerURL))))
		})

		It("uses the space repo to create the space", func() {
			Expect(spaceRepo.CreateSpaceCallCount()).To(Equal(1))
			_, spaceRecord := spaceRepo.CreateSpaceArgsForCall(0)
			Expect(spaceRecord.Name).To(Equal("the-space"))
		})

		When("a field in the request has invalid value", func() {
			BeforeEach(func() {
				requestBody = `{
                    "name": 123,
                    "relationships": {
                        "organization": {
                            "data": {
                                "guid": "[org-guid]"
                            }
                        }
                    }
                }`
			})

			It("returns a bad request error", func() {
				expectUnprocessableEntityError("Name must be a string")
			})
		})

		When("the request body is invalid json", func() {
			BeforeEach(func() {
				requestBody = `{"definitely not valid json`
			})

			It("returns a bad request error", func() {
				expectBadRequestError()
			})
		})

		When("when a required field is not provided", func() {
			BeforeEach(func() {
				requestBody = `{
                    "name": "the-name",
                    "relationships": {
                    }
                }`
			})

			It("returns a bad request error", func() {
				expectUnprocessableEntityError("Data is a required field")
			})
		})

		When("the space repo returns a uniqueness error", func() {
			BeforeEach(func() {
				var err error = &k8serrors.StatusError{
					ErrStatus: metav1.Status{
						Reason: metav1.StatusReason(fmt.Sprintf(`{"code":%d}`, workloads.DuplicateSpaceNameError)),
					},
				}
				spaceRepo.CreateSpaceReturns(repositories.SpaceRecord{}, err)
			})

			It("returns an error", func() {
				expectUnprocessableEntityError("Space 'the-space' already exists.")
			})
		})

		When("the space repo returns another error", func() {
			BeforeEach(func() {
				spaceRepo.CreateSpaceReturns(repositories.SpaceRecord{}, errors.New("boom"))
			})

			It("returns unknown error", func() {
				expectUnknownError()
			})
		})
	})

	Describe("Listing Spaces", func() {
		BeforeEach(func() {
			requestMethod = http.MethodGet
			spaceRepo.FetchSpacesReturns([]repositories.SpaceRecord{
				{
					Name:             "alice",
					GUID:             "a-l-i-c-e",
					OrganizationGUID: "org-guid-1",
					CreatedAt:        now,
					UpdatedAt:        now,
				},
				{
					Name:             "bob",
					GUID:             "b-o-b",
					OrganizationGUID: "org-guid-2",
					CreatedAt:        now,
					UpdatedAt:        now,
				},
			}, nil)
		})

		It("returns a list of spaces", func() {
			Expect(rr.Header().Get("Content-Type")).To(Equal("application/json"))

			Expect(rr.Body.String()).To(MatchJSON(fmt.Sprintf(`{
                "pagination": {
                    "total_results": 2,
                    "total_pages": 1,
                    "first": {
                        "href": "%[1]s/v3/spaces?page=1"
                    },
                    "last": {
                        "href": "%[1]s/v3/spaces?page=1"
                    },
                    "next": null,
                    "previous": null
                },
                "resources": [
                {
                    "guid": "a-l-i-c-e",
                    "name": "alice",
                    "created_at": "2021-09-17T15:23:10Z",
                    "updated_at": "2021-09-17T15:23:10Z",
                    "metadata": {
                        "labels": {},
                        "annotations": {}
                    },
                    "relationships": {
                        "organization": {
                            "data": {
                                "guid": "org-guid-1"
                            }
                        }
                    },
                    "links": {
                        "self": {
                            "href": "%[1]s/v3/spaces/a-l-i-c-e"
                        },
                        "organization": {
                            "href": "%[1]s/v3/organizations/org-guid-1"
                        }
                    }
                },
                {
                    "guid": "b-o-b",
                    "name": "bob",
                    "created_at": "2021-09-17T15:23:10Z",
                    "updated_at": "2021-09-17T15:23:10Z",
                    "metadata": {
                        "labels": {},
                        "annotations": {}
                    },
                    "relationships": {
                        "organization": {
                            "data": {
                                "guid": "org-guid-2"
                            }
                        }
                    },
                    "links": {
                        "self": {
                            "href": "%[1]s/v3/spaces/b-o-b"
                        },
                        "organization": {
                            "href": "%[1]s/v3/organizations/org-guid-2"
                        }
                    }
                }
                ]
            }`, defaultServerURL)))

			Expect(spaceRepo.FetchSpacesCallCount()).To(Equal(1))
			_, organizationGUIDs, names := spaceRepo.FetchSpacesArgsForCall(0)
			Expect(organizationGUIDs).To(BeEmpty())
			Expect(names).To(BeEmpty())
		})

		When("fetching the spaces fails", func() {
			BeforeEach(func() {
				spaceRepo.FetchSpacesReturns(nil, errors.New("boom!"))
			})

			It("returns an error", func() {
				expectUnknownError()
			})
		})

		When("organization_guids are provided as a comma-separated list", func() {
			BeforeEach(func() {
				requestPath = spacesBase + "?organization_guids=foo,,bar,"
			})

			It("filters spaces by them", func() {
				Expect(spaceRepo.FetchSpacesCallCount()).To(Equal(1))
				_, organizationGUIDs, names := spaceRepo.FetchSpacesArgsForCall(0)
				Expect(organizationGUIDs).To(ConsistOf("foo", "bar"))
				Expect(names).To(BeEmpty())
			})
		})

		When("names are provided as a comma-separated list", func() {
			BeforeEach(func() {
				requestPath = spacesBase + "?organization_guids=org1&names=foo,,bar,"
			})

			It("filters spaces by them", func() {
				Expect(spaceRepo.FetchSpacesCallCount()).To(Equal(1))
				_, organizationGUIDs, names := spaceRepo.FetchSpacesArgsForCall(0)
				Expect(organizationGUIDs).To(ConsistOf("org1"))
				Expect(names).To(ConsistOf("foo", "bar"))
			})
		})
	})
})
