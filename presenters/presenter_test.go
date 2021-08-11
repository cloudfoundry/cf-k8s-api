package presenters_test

import (
	"cloudfoundry.org/cf-k8s-api/presenters"
	"encoding/json"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"testing"
)

func TestRootPresenters(t *testing.T) {
	spec.Run(t, "object", testRootV3Presenter, spec.Report(report.Terminal{}))
}

func testRootV3Presenter(t *testing.T, when spec.G, it spec.S) {

	Expect := NewWithT(t).Expect
	// This is just for the gomega BeAssignableToTypeOf test below- works like type assertion
	var dummyString string

	it("contains expected fields with correct types", func() {
		var rootV3PresenterSelf = presenters.RootV3PresenterSelf{}
		Expect(rootV3PresenterSelf).To(MatchAllFields(Fields{
			"Href": BeAssignableToTypeOf(dummyString),
		}))

		var rootV3PresenterLinks = presenters.RootV3PresenterLinks{}
		Expect(rootV3PresenterLinks).To(MatchAllFields(Fields{
			"Self": BeAssignableToTypeOf(rootV3PresenterSelf),
		}))

		var rootV3Presenter = presenters.RootV3Presenter{}
		Expect(rootV3Presenter).To(MatchAllFields(Fields{
			"Links": BeAssignableToTypeOf(rootV3PresenterLinks),
		}))
	})

	it( "decodes to JSON", func() {
		var rootV3Presenter = presenters.RootV3Presenter{}
		_, err := json.Marshal(rootV3Presenter)
		Expect(err).To(BeNil())
	})




}
