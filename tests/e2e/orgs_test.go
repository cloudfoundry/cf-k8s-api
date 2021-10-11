package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/form3tech-oss/jwt-go"
	"github.com/go-http-utils/headers"
	. "github.com/onsi/gomega/gstruct"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"code.cloudfoundry.org/cf-k8s-api/presenter"
	"code.cloudfoundry.org/cf-k8s-api/repositories"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Orgs", func() {
	Describe("creating orgs", func() {
		var orgName string

		BeforeEach(func() {
			orgName = generateGUID("org")
		})

		AfterEach(func() {
			deleteSubnamespaceByLabel(rootNamespace, orgName)
		})

		It("creates an org", func() {
			orgsUrl := apiServerRoot + "/v3/organizations"

			body := fmt.Sprintf(`{ "name": "%s" }`, orgName)
			req, err := http.NewRequest(http.MethodPost, orgsUrl, strings.NewReader(body))
			Expect(err).NotTo(HaveOccurred())

			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusCreated))
			Expect(resp.Header["Content-Type"]).To(ConsistOf("application/json"))
			responseMap := map[string]interface{}{}
			Expect(json.NewDecoder(resp.Body).Decode(&responseMap)).To(Succeed())
			Expect(responseMap["name"]).To(Equal(orgName))
		})
	})

	Describe("listing orgs", func() {
		var (
			orgs       []hierarchicalNamespace
			userName   string
			authHeader string
		)

		BeforeEach(func() {
			userName = "alice"
			authHeader = fmt.Sprintf("Bearer: %s", generateJWTToken(userName))
			orgs = []hierarchicalNamespace{}
			for i := 1; i < 4; i++ {
				org := createOrg(generateGUID("org" + strconv.Itoa(i)))
				bindUserToOrg(userName, org)
				orgs = append(orgs, org)
			}

			orgs = append(orgs, createOrg(generateGUID("org4")))
		})

		AfterEach(func() {
			for _, org := range orgs {
				deleteSubnamespace(rootNamespace, org.guid)
			}
		})

		It("returns all 3 orgs that alice has a role in", func() {
			Eventually(getOrgsFn(authHeader)).Should(ContainElements(
				MatchFields(IgnoreExtras, Fields{"Name": Equal(orgs[0].label)}),
				MatchFields(IgnoreExtras, Fields{"Name": Equal(orgs[1].label)}),
				MatchFields(IgnoreExtras, Fields{"Name": Equal(orgs[2].label)}),
			))
		})

		It("does not return orgs alice does not have a role in", func() {
			Consistently(getOrgsFn(authHeader)).ShouldNot(ContainElements(
				MatchFields(IgnoreExtras, Fields{"Name": Equal(orgs[3].label)}),
			))
		})

		When("org names are filtered", func() {
			It("returns orgs 1 & 3", func() {
				Eventually(getOrgsFn(authHeader, orgs[0].label, orgs[2].label)).Should(ContainElements(
					MatchFields(IgnoreExtras, Fields{"Name": Equal(orgs[0].label)}),
					MatchFields(IgnoreExtras, Fields{"Name": Equal(orgs[2].label)}),
				))
				Consistently(getOrgsFn(authHeader, orgs[0].label, orgs[2].label), "2s").ShouldNot(ContainElement(
					MatchFields(IgnoreExtras, Fields{"Name": Equal(orgs[1].label)}),
				))
			})
		})

		When("no Authorization header is available in the request", func() {
			It("returns unauthorized error", func() {
				orgsUrl := apiServerRoot + "/v3/organizations"
				req, err := http.NewRequest(http.MethodGet, orgsUrl, nil)
				Expect(err).NotTo(HaveOccurred())
				resp, err := http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
			})
		})
	})
})

func getOrgsFn(authHeaderValue string, names ...string) func() ([]presenter.OrgResponse, error) {
	return func() ([]presenter.OrgResponse, error) {
		orgsUrl := apiServerRoot + "/v3/organizations"

		if len(names) > 0 {
			orgsUrl += "?names=" + strings.Join(names, ",")
		}

		req, err := http.NewRequest(http.MethodGet, orgsUrl, nil)
		if err != nil {
			return nil, err
		}

		if authHeaderValue != "" {
			req.Header.Add(headers.Authorization, authHeaderValue)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("bad status: %d", resp.StatusCode)
		}

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		orgList := presenter.OrgListResponse{}
		err = json.Unmarshal(bodyBytes, &orgList)
		if err != nil {
			return nil, err
		}

		return orgList.Resources, nil
	}
}

func createOrg(orgName string) hierarchicalNamespace {
	orgDetails := createHierarchicalNamespace(rootNamespace, orgName, repositories.OrgNameLabel)
	waitForSubnamespaceAnchor(rootNamespace, orgDetails.guid)
	return orgDetails
}

func bindUserToOrg(userName string, org hierarchicalNamespace) {
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: org.guid,
			Name:      userName + "-spacedeveloper",
		},
		Subjects: []rbacv1.Subject{{Kind: rbacv1.UserKind, Name: userName}},
		RoleRef:  rbacv1.RoleRef{Kind: "ClusterRole", Name: "cf-admin-clusterrolebinding"},
	}
	Expect(k8sClient.Create(context.Background(), roleBinding)).To(Succeed())
}

func generateJWTToken(userName string) string {
	atClaims := jwt.MapClaims{}
	atClaims["sub"] = userName
	atClaims["exp"] = time.Now().Add(time.Minute * 15).Unix()
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims)
	token, err := at.SignedString([]byte("jwt-token-for-" + userName))
	Expect(err).NotTo(HaveOccurred())
	return token
}
