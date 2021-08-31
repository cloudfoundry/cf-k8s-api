package apis

import (
	"bytes"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestApps(t *testing.T) {
	spec.Run(t, "AppsCreateHandler", testAppsGetHandler, spec.Report(report.Terminal{}))
}

func testAppsGetHandler(t *testing.T, when spec.G, it spec.S) {
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

				apiHandler := AppHandler{
					ServerURL: defaultServerURL,
				}

				rr = httptest.NewRecorder()
				handler := http.HandlerFunc(apiHandler.AppsCreateHandler)
				handler.ServeHTTP(rr, req)

			})
			it("returns a status 400 Bad Request ", func() {
				Expect(rr.Code).To(Equal(http.StatusBadRequest))
			})
			it("has the expected error response body", func() {
				Expect(rr.Body.Bytes()).NotTo(BeEmpty())
			})

		})

	})
}
