package authorization_test

import (
	"code.cloudfoundry.org/cf-k8s-api/repositories/authorization"
	"code.cloudfoundry.org/cf-k8s-api/repositories/authorization/fake"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Org", func() {
	var (
		org               *authorization.Org
		identityInspector *fake.IdentityInspector
	)

	BeforeEach(func() {
		identityInspector = new(fake.IdentityInspector)
		org = authorization.NewOrg(k8sClient, identityInspector)
	})

	It("foos", func() {
		Expect(org).To(BeNil())
	})
})
