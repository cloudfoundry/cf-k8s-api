package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-http-utils/headers"
	. "github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"code.cloudfoundry.org/cf-k8s-api/presenter"
	"code.cloudfoundry.org/cf-k8s-api/repositories"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Orgs", func() {
	var (
		serviceAccountName string
		authHeader         string
	)

	createOrgWithHeaders := func(orgName string, headers map[string]string) (*http.Response, error) {
		orgsUrl := apiServerRoot + "/v3/organizations"
		body := fmt.Sprintf(`{ "name": "%s" }`, orgName)
		req, err := http.NewRequest(http.MethodPost, orgsUrl, strings.NewReader(body))
		Expect(err).NotTo(HaveOccurred())

		for key, value := range headers {
			req.Header.Add(key, value)
		}
		return http.DefaultClient.Do(req)
	}

	BeforeEach(func() {
		serviceAccountName = generateGUID("user")
		token := obtainServiceAccountToken(serviceAccountName)
		authHeader = fmt.Sprintf("Bearer %s", token)
	})

	AfterEach(func() {
		deleteServiceAccount(serviceAccountName)
	})

	Describe("creating orgs", func() {
		var orgName string

		BeforeEach(func() {
			orgName = generateGUID("org")
		})

		AfterEach(func() {
			deleteSubnamespaceByLabel(rootNamespace, orgName)
		})

		It("creates an org", func() {
			resp, err := createOrgWithHeaders(orgName, map[string]string{headers.Authorization: authHeader})
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusCreated))
			Expect(resp.Header["Content-Type"]).To(ConsistOf("application/json"))
			responseMap := map[string]interface{}{}
			Expect(json.NewDecoder(resp.Body).Decode(&responseMap)).To(Succeed())
			Expect(responseMap["name"]).To(Equal(orgName))

			nsName, ok := responseMap["guid"].(string)
			Expect(ok).To(BeTrue())
			Expect(k8sClient.Get(context.Background(), client.ObjectKey{Name: nsName}, &corev1.Namespace{})).To(Succeed())
		})

		When("the org name already exists", func() {
			BeforeEach(func() {
				resp, err := createOrgWithHeaders(orgName, map[string]string{headers.Authorization: authHeader})
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				Expect(resp.StatusCode).To(Equal(http.StatusCreated))
			})

			It("returns an unprocessable entity error", func() {
				resp, err := createOrgWithHeaders(orgName, map[string]string{headers.Authorization: authHeader})
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				Expect(resp).To(HaveHTTPStatus(http.StatusUnprocessableEntity))
				responseMap := map[string]interface{}{}
				Expect(json.NewDecoder(resp.Body).Decode(&responseMap)).To(Succeed())
				Expect(responseMap).To(HaveKeyWithValue("errors", BeAssignableToTypeOf([]interface{}{})))
				errs := responseMap["errors"].([]interface{})
				Expect(errs[0]).To(SatisfyAll(
					HaveKeyWithValue("code", BeNumerically("==", 10008)),
					HaveKeyWithValue("detail", MatchRegexp(fmt.Sprintf(`Organization '%s' already exists.`, orgName))),
					HaveKeyWithValue("title", Equal("CF-UnprocessableEntity")),
				))
			})
		})
	})

	Describe("listing orgs", func() {
		var orgs []hierarchicalNamespace

		BeforeEach(func() {
			orgs = []hierarchicalNamespace{}
			for i := 1; i < 4; i++ {
				org := createOrgNamespace(generateGUID("org" + strconv.Itoa(i)))
				bindServiceAccountToOrg(serviceAccountName, org)
				orgs = append(orgs, org)
			}

			orgs = append(orgs, createOrgNamespace(generateGUID("org4")))
		})

		AfterEach(func() {
			for _, org := range orgs {
				deleteSubnamespace(rootNamespace, org.guid)
			}
		})

		It("returns all 3 orgs that the service account has a role in", func() {
			Eventually(getOrgsFn(authHeader)).Should(ContainElements(
				MatchFields(IgnoreExtras, Fields{"Name": Equal(orgs[0].label)}),
				MatchFields(IgnoreExtras, Fields{"Name": Equal(orgs[1].label)}),
				MatchFields(IgnoreExtras, Fields{"Name": Equal(orgs[2].label)}),
			))
		})

		It("does not return orgs the service account does not have a role in", func() {
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

func createOrgNamespace(orgName string) hierarchicalNamespace {
	orgDetails := createHierarchicalNamespace(rootNamespace, orgName, repositories.OrgNameLabel)
	waitForSubnamespaceAnchor(rootNamespace, orgDetails.guid)
	return orgDetails
}

func bindServiceAccountToOrg(userName string, org hierarchicalNamespace) {
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: org.guid,
			Name:      userName + "-admin",
		},
		Subjects: []rbacv1.Subject{{Kind: rbacv1.ServiceAccountKind, Name: userName}},
		RoleRef:  rbacv1.RoleRef{Kind: "ClusterRole", Name: "cf-admin-clusterrole"},
	}
	Expect(k8sClient.Create(context.Background(), roleBinding)).To(Succeed())
}

func obtainServiceAccountToken(name string) string {
	var err error

	serviceAccount := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: rootNamespace,
		},
	}
	err = k8sClient.Create(context.Background(), &serviceAccount)
	Expect(err).NotTo(HaveOccurred())

	Eventually(func() error {
		if err := k8sClient.Get(context.Background(), client.ObjectKeyFromObject(&serviceAccount), &serviceAccount); err != nil {
			return err
		}

		if len(serviceAccount.Secrets) != 1 {
			return fmt.Errorf("expected exactly 1 secret, got %d", len(serviceAccount.Secrets))
		}

		return nil
	}, "30s").Should(Succeed())

	tokenSecret := corev1.Secret{}
	Eventually(func() error {
		return k8sClient.Get(context.Background(), client.ObjectKey{Name: serviceAccount.Secrets[0].Name, Namespace: rootNamespace}, &tokenSecret)
	}).Should(Succeed())

	return string(tokenSecret.Data["token"])
}

func deleteServiceAccount(name string) {
	serviceAccount := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: rootNamespace,
		},
	}

	Expect(k8sClient.Delete(context.Background(), &serviceAccount)).To(Succeed())
}
