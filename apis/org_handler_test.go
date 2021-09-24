package apis_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"code.cloudfoundry.org/cf-k8s-api/apis"
	"code.cloudfoundry.org/cf-k8s-api/apis/fake"
	"code.cloudfoundry.org/cf-k8s-api/repositories"
	"github.com/gorilla/mux"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

const (
	rootURL = "https://api.example.org"
)

func TestOrg(t *testing.T) {
	spec.Run(t, "listing orgs", testListingOrgs, spec.Report(report.Terminal{}))
	spec.Run(t, "listing spaces", testListingSpaces, spec.Report(report.Terminal{}))
}

var (
	now = time.Unix(1631892190, 0) // 2021-09-17T15:23:10Z

	ctx        context.Context
	router     *mux.Router
	orgHandler *apis.OrgHandler
	orgRepo    *fake.CFOrgRepository
	req        *http.Request
	rr         *httptest.ResponseRecorder
	err        error
)

func testListingOrgs(t *testing.T, when spec.G, it spec.S) {
	const orgsBase = "/v3/organizations"

	g := NewWithT(t)

	it.Before(func() {
		ctx = context.Background()
		orgRepo = new(fake.CFOrgRepository)

		orgRepo.FetchOrgsReturns([]repositories.OrgRecord{
			{
				Name:      "alice",
				GUID:      "a-l-i-c-e",
				CreatedAt: now,
				UpdatedAt: now,
			},
			{
				Name:      "bob",
				GUID:      "b-o-b",
				CreatedAt: now,
				UpdatedAt: now,
			},
		}, nil)

		orgHandler = apis.NewOrgHandler(orgRepo, rootURL)
		router = mux.NewRouter()
		orgHandler.RegisterRoutes(router)

		rr = httptest.NewRecorder()
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, orgsBase, nil)
		g.Expect(err).NotTo(HaveOccurred())
	})

	when("happy path", func() {
		it.Before(func() {
			router.ServeHTTP(rr, req)
		})

		it("returns 200", func() {
			g.Expect(rr.Result().StatusCode).To(Equal(http.StatusOK))
		})

		it("sets json content type", func() {
			g.Expect(rr.Header().Get("Content-Type")).To(Equal("application/json"))
		})

		it("lists orgs using the repository", func() {
			g.Expect(orgRepo.FetchOrgsCallCount()).To(Equal(1))
			_, names := orgRepo.FetchOrgsArgsForCall(0)
			g.Expect(names).To(BeEmpty())
		})

		it("renders the orgs response", func() {
			expectedBody := fmt.Sprintf(`
                {
                   "pagination": {
                      "total_results": 2,
                      "total_pages": 1,
                      "first": {
                         "href": "%[1]s/v3/organizations?page=1"
                      },
                      "last": {
                         "href": "%[1]s/v3/organizations?page=1"
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
                            "suspended": false,
                            "metadata": {
                              "labels": {},
                              "annotations": {}
                            },
                            "relationships": {},
                            "links": {
                                "self": {
                                    "href": "%[1]s/v3/organizations/a-l-i-c-e"
                                }
                            }
                        },
                        {
                            "guid": "b-o-b",
                            "name": "bob",
                            "created_at": "2021-09-17T15:23:10Z",
                            "updated_at": "2021-09-17T15:23:10Z",
                            "suspended": false,
                            "metadata": {
                              "labels": {},
                              "annotations": {}
                            },
                            "relationships": {},
                            "links": {
                                "self": {
                                    "href": "%[1]s/v3/organizations/b-o-b"
                                }
                            }
                        }
                    ]
                }`, rootURL)
			g.Expect(rr.Body.String()).To(MatchJSON(expectedBody))
		})
	})

	when("names are specified", func() {
		it.Before(func() {
			req, err = http.NewRequestWithContext(ctx, http.MethodGet, orgsBase+"?names=foo,bar", nil)
			g.Expect(err).NotTo(HaveOccurred())

			router.ServeHTTP(rr, req)
		})

		it("filters by them", func() {
			g.Expect(orgRepo.FetchOrgsCallCount()).To(Equal(1))
			_, names := orgRepo.FetchOrgsArgsForCall(0)
			g.Expect(names).To(ConsistOf("foo", "bar"))
		})
	})

	when("fetching the orgs fails", func() {
		it.Before(func() {
			orgRepo.FetchOrgsReturns(nil, errors.New("boom!"))
			router.ServeHTTP(rr, req)
		})

		itRespondsWithUnknownError(it, g, func() *httptest.ResponseRecorder { return rr })
	})
}

func testListingSpaces(t *testing.T, when spec.G, it spec.S) {
	const spacesBase = "/v3/spaces"

	g := NewWithT(t)

	it.Before(func() {
		ctx = context.Background()
		orgRepo = new(fake.CFOrgRepository)

		orgRepo.FetchSpacesReturns([]repositories.SpaceRecord{
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

		orgHandler = apis.NewOrgHandler(orgRepo, rootURL)
		router = mux.NewRouter()
		orgHandler.RegisterRoutes(router)

		rr = httptest.NewRecorder()
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, spacesBase, nil)
		g.Expect(err).NotTo(HaveOccurred())
	})

	it("returns a list of spaces", func() {
		router.ServeHTTP(rr, req)
		g.Expect(rr.Header().Get("Content-Type")).To(Equal("application/json"))

		g.Expect(rr.Body.String()).To(MatchJSON(fmt.Sprintf(`{
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
        }`, rootURL)))

		g.Expect(orgRepo.FetchSpacesCallCount()).To(Equal(1))
		_, organizationGUIDs, names := orgRepo.FetchSpacesArgsForCall(0)
		g.Expect(organizationGUIDs).To(BeEmpty())
		g.Expect(names).To(BeEmpty())
	})

	when("fetching the spaces fails", func() {
		it.Before(func() {
			orgRepo.FetchSpacesReturns(nil, errors.New("boom!"))
			router.ServeHTTP(rr, req)
		})

		itRespondsWithUnknownError(it, g, func() *httptest.ResponseRecorder { return rr })
	})

	when("organization_guids are provided as a comma-separated list", func() {
		it("filters spaces by them", func() {
			req, err = http.NewRequestWithContext(ctx, http.MethodGet, spacesBase+"?organization_guids=foo,,bar,", nil)
			g.Expect(err).NotTo(HaveOccurred())
			router.ServeHTTP(rr, req)

			g.Expect(orgRepo.FetchSpacesCallCount()).To(Equal(1))
			_, organizationGUIDs, names := orgRepo.FetchSpacesArgsForCall(0)
			g.Expect(organizationGUIDs).To(ConsistOf("foo", "bar"))
			g.Expect(names).To(BeEmpty())
		})
	})

	when("names are provided as a comma-separated list", func() {
		it("filters spaces by them", func() {
			req, err = http.NewRequestWithContext(ctx, http.MethodGet, spacesBase+"?organization_guids=org1&names=foo,,bar,", nil)
			g.Expect(err).NotTo(HaveOccurred())
			router.ServeHTTP(rr, req)

			g.Expect(orgRepo.FetchSpacesCallCount()).To(Equal(1))
			_, organizationGUIDs, names := orgRepo.FetchSpacesArgsForCall(0)
			g.Expect(organizationGUIDs).To(ConsistOf("org1"))
			g.Expect(names).To(ConsistOf("foo", "bar"))
		})
	})
}
