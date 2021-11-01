package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"code.cloudfoundry.org/cf-k8s-api/presenter"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Roles", func() {
	var (
		ctx      context.Context
		userName string
	)

	createRoleWithHeaders := func(roleName, userName, spaceGUID string, headers map[string]string) (*http.Response, error) {
		rolesURL := apiServerRoot + "/v3/roles"
		body := fmt.Sprintf(`{
          "type": "%s",
          "relationships": {
            "user": {
              "data": {
                "guid": "%s"
              }
            },
            "space": {
              "data": {
                "guid": "%s"
              }
            }
          }
        }`, roleName, userName, spaceGUID)
		req, err := http.NewRequest(http.MethodPost, rolesURL, strings.NewReader(body))
		Expect(err).NotTo(HaveOccurred())

		for key, value := range headers {
			req.Header.Add(key, value)
		}
		return http.DefaultClient.Do(req)
	}

	BeforeEach(func() {
		ctx = context.Background()
		userName = uuid.NewString()
	})

	Describe("creating a role", func() {
		var (
			org   presenter.OrgResponse
			space presenter.SpaceResponse
		)

		BeforeEach(func() {
			org = createOrg(uuid.NewString())
			space = createSpace(uuid.NewString(), org.GUID)
		})

		AfterEach(func() {
			deleteSubnamespace(org.GUID, space.GUID)
			deleteSubnamespace(rootNamespace, org.GUID)
		})

		FIt("does something", func() {
			response, err := createRoleWithHeaders("space_developer", userName, space.GUID, nil)
			Expect(err).NotTo(HaveOccurred())

			Expect(response).To(HaveHTTPStatus(http.StatusCreated))

			defer response.Body.Close()

			responseMap := map[string]interface{}{}
			Expect(json.NewDecoder(response.Body).Decode(&responseMap)).To(Succeed())

			Expect(responseMap).To(HaveKeyWithValue("type", "space_developer"))

			roleBindingList := &rbacv1.RoleBindingList{}
			Eventually(func() ([]rbacv1.RoleBinding, error) {
				err := k8sClient.List(ctx, roleBindingList, client.InNamespace(space.GUID))
				if err != nil {
					return nil, err
				}
				return roleBindingList.Items, nil
			}).Should(HaveLen(1))

			binding := roleBindingList.Items[0]
			Expect(responseMap).To(HaveKeyWithValue("guid", binding.UID))
			Expect(binding.RoleRef.Name).To(Equal("space_developer"))
			Expect(binding.RoleRef.Kind).To(Equal("ClusterRole"))
			Expect(binding.Subjects).To(HaveLen(1))
			subject := binding.Subjects[0]
			Expect(subject.Name).To(Equal(userName))
			Expect(subject.Kind).To(Equal("User"))
		})
	})
})
