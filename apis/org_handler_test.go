package apis_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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
	orgsBase = "/v3/organizations"
)

func TestOrg(t *testing.T) {
	spec.Run(t, "listing orgs", testListingOrgs, spec.Report(report.Terminal{}))
	spec.Run(t, "creating orgs", testCreateOrg, spec.Report(report.Terminal{}))
}

func testCreateOrg(t *testing.T, when spec.G, it spec.S) {
	var (
		now        = time.Unix(1631892190, 0) // 2021-09-17T15:23:10Z
		ctx        context.Context
		router     *mux.Router
		orgHandler *apis.OrgHandler
		orgRepo    *fake.CFOrgRepository
		rr         *httptest.ResponseRecorder
	)

	g := NewWithT(t)

	makePostRequest := func(requestBody string) {
		req, err := http.NewRequestWithContext(ctx, "POST", orgsBase, strings.NewReader(requestBody))
		g.Expect(err).NotTo(HaveOccurred())

		router.ServeHTTP(rr, req)
	}

	it.Before(func() {
		ctx = context.Background()
		orgRepo = new(fake.CFOrgRepository)
		orgRepo.CreateOrgStub = func(_ context.Context, record repositories.OrgRecord) (repositories.OrgRecord, error) {
			record.GUID = "t-h-e-o-r-g"
			record.CreatedAt = now
			record.UpdatedAt = now
			return record, nil
		}

		serverURL, err := url.Parse(defaultServerURL)
		g.Expect(err).NotTo(HaveOccurred())

		orgHandler = apis.NewOrgHandler(orgRepo, *serverURL)
		router = mux.NewRouter()
		orgHandler.RegisterRoutes(router)

		rr = httptest.NewRecorder()
	})

	when("happy path", func() {
		it.Before(func() {
			makePostRequest(`{"name": "the-org"}`)
		})

		it("returns 201 with appropriate success JSON", func() {
			g.Expect(rr).To(HaveHTTPStatus(http.StatusCreated))
			g.Expect(rr).To(HaveHTTPHeaderWithValue("Content-Type", "application/json"))
			g.Expect(rr).To(HaveHTTPBody(MatchJSON(fmt.Sprintf(`{
"guid": "t-h-e-o-r-g",
					"name": "the-org",
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
							"href": "%[1]s/v3/organizations/t-h-e-o-r-g"
						}
					}
				}`, defaultServerURL))))
		})

		it("invokes the repo org create function with expected parameters", func() {
			g.Expect(orgRepo.CreateOrgCallCount()).To(Equal(1))
			_, orgRecord := orgRepo.CreateOrgArgsForCall(0)
			g.Expect(orgRecord.Name).To(Equal("the-org"))
			g.Expect(orgRecord.Suspended).To(BeFalse())
			g.Expect(orgRecord.Labels).To(BeEmpty())
			g.Expect(orgRecord.Annotations).To(BeEmpty())
		})
	})

	when("the org repo returns an error", func() {
		it.Before(func() {
			orgRepo.CreateOrgReturns(repositories.OrgRecord{}, errors.New("boom"))
			makePostRequest(`{"name": "the-org"}`)
		})

		itRespondsWithUnknownError(it, g, func() *httptest.ResponseRecorder { return rr })
	})

	when("the user passes optional org parameters", func() {
		it.Before(func() {
			makePostRequest(`{
"name": "the-org",
					"suspended": true,
					"metadata": {
						"labels": {"foo": "bar"},
						"annotations": {"bar": "baz"}
					}
				}`)
		})

		it("invokes the repo org create function with expected parameters", func() {
			g.Expect(rr).To(HaveHTTPStatus(http.StatusCreated))
			g.Expect(rr).To(HaveHTTPHeaderWithValue("Content-Type", "application/json"))
			g.Expect(orgRepo.CreateOrgCallCount()).To(Equal(1))
			_, orgRecord := orgRepo.CreateOrgArgsForCall(0)
			g.Expect(orgRecord.Name).To(Equal("the-org"))
			g.Expect(orgRecord.Suspended).To(BeTrue())
			g.Expect(orgRecord.Labels).To(And(HaveLen(1), HaveKeyWithValue("foo", "bar")))
			g.Expect(orgRecord.Annotations).To(And(HaveLen(1), HaveKeyWithValue("bar", "baz")))
		})

		it("returns 201 with appropriate success JSON", func() {
			g.Expect(rr).To(HaveHTTPStatus(http.StatusCreated))
			g.Expect(rr).To(HaveHTTPHeaderWithValue("Content-Type", "application/json"))
			g.Expect(rr).To(HaveHTTPBody(MatchJSON(fmt.Sprintf(`{
"guid": "t-h-e-o-r-g",
					"name": "the-org",
					"created_at": "2021-09-17T15:23:10Z",
					"updated_at": "2021-09-17T15:23:10Z",
					"suspended": true,
					"metadata": {
					  "labels": {"foo": "bar"},
					  "annotations": {"bar": "baz"}
					},
					"relationships": {},
					"links": {
						"self": {
							"href": "%[1]s/v3/organizations/t-h-e-o-r-g"
						}
					}
				}`, defaultServerURL))))
		})
	})

	when("the request body is invalid json", func() {
		it.Before(func() {
			makePostRequest(`{`)
		})

		it("returns a status 400 with appropriate error JSON", func() {
			g.Expect(rr).To(HaveHTTPStatus(http.StatusBadRequest))
			g.Expect(rr).To(HaveHTTPHeaderWithValue("Content-Type", "application/json"))
			g.Expect(rr).To(HaveHTTPBody(MatchJSON(`{
"errors": [
					{
						"title": "CF-MessageParseError",
						"detail": "Request invalid due to parse error: invalid request body",
						"code": 1001
					}
				]
			}`)))
		})
	})

	when("the request body has an unknown field", func() {
		it.Before(func() {
			makePostRequest(`{"description" : "Invalid Request"}`)
		})

		it("returns a status 422 with appropriate error JSON", func() {
			g.Expect(rr).To(HaveHTTPStatus(http.StatusUnprocessableEntity))
			g.Expect(rr).To(HaveHTTPHeaderWithValue("Content-Type", "application/json"))
			g.Expect(rr).To(HaveHTTPBody(MatchJSON(`{
"errors": [
					{
						"title": "CF-UnprocessableEntity",
						"detail": "invalid request body: json: unknown field \"description\"",
						"code": 10008
					}
				]
			}`)))
		})
	})

	when("the request body is invalid with invalid app name", func() {
		it.Before(func() {
			makePostRequest(`{"name": 12345}`)
		})

		it("returns a status 422 with appropriate error JSON", func() {
			g.Expect(rr).To(HaveHTTPStatus(http.StatusUnprocessableEntity))
			g.Expect(rr).To(HaveHTTPHeaderWithValue("Content-Type", "application/json"))
			g.Expect(rr).To(HaveHTTPBody(MatchJSON(`{
"errors": [
					{
						"code":   10008,
						"title": "CF-UnprocessableEntity",
						"detail": "Name must be a string"
					}
				]
			}`)))
		})
	})

	when("the request body is invalid with missing required name field", func() {
		it.Before(func() {
			makePostRequest(`{"metadata": {"labels": {"foo": "bar"}}}`)
		})

		it("returns a status 422 with appropriate error message json", func() {
			g.Expect(rr).To(HaveHTTPStatus(http.StatusUnprocessableEntity))
			g.Expect(rr).To(HaveHTTPHeaderWithValue("Content-Type", "application/json"))
			g.Expect(rr).To(HaveHTTPBody(MatchJSON(`{
"errors": [
					{
						"title": "CF-UnprocessableEntity",
						"detail": "Name is a required field",
						"code": 10008
					}
				]
			}`)))
		})
	})
}

func testListingOrgs(t *testing.T, when spec.G, it spec.S) {
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

		serverURL, err := url.Parse(defaultServerURL)
		g.Expect(err).NotTo(HaveOccurred())

		orgHandler = apis.NewOrgHandler(orgRepo, *serverURL)
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
				}`, defaultServerURL)
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
