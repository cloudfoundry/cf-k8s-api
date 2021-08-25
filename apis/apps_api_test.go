package apis_test

// import (
// 	"net/http"
// 	"net/http/httptest"
// 	"testing"

// 	"github.com/sclevine/spec/report"

// 	"code.cloudfoundry.org/cf-k8s-api/apis"
// 	. "github.com/onsi/gomega"
// 	"github.com/sclevine/spec"
// )

// func TestApps(t *testing.T) {
// 	spec.Run(t, "object", testAppsGetHandler, spec.Report(report.Terminal{}))
// }

// func testAppsGetHandler(t *testing.T, when spec.G, it spec.S) {
// 	Expect := NewWithT(t).Expect

// 	when("the GET /v3/apps/:guid  endpoint returns successfully", func() {
// 		var rr *httptest.ResponseRecorder
// 		it.Before(func() {
// 			// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
// 			// pass 'nil' as the third parameter.
// 			req, err := http.NewRequest("GET", "/v3/apps/my-app-guid", nil)
// 			Expect(err).NotTo(HaveOccurred())

// 			// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
// 			rr = httptest.NewRecorder()
// 			apiHandler := apis.AppsHandler{
// 				ServerURL: defaultServerURL,
// 			}

// 			handler := http.HandlerFunc(apiHandler.AppsGetHandler)

// 			// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
// 			// directly and pass in our Request and ResponseRecorder.
// 			handler.ServeHTTP(rr, req)
// 		})

// 		it("returns status 200 OK", func() {
// 			httpStatus := rr.Code
// 			Expect(httpStatus).Should(Equal(http.StatusOK), "Matching HTTP response code:")
// 		})

// 		it("returns Content-Type as JSON in header", func() {
// 			contentTypeHeader := rr.Header().Get("Content-Type")
// 			Expect(contentTypeHeader).Should(Equal(jsonHeader), "Matching Content-Type header:")
// 		})
// 	})
// }
