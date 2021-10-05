//go:build e2e
// +build e2e

package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"testing"

	. "github.com/onsi/gomega/gstruct"

	"code.cloudfoundry.org/cf-k8s-api/presenter"
	"code.cloudfoundry.org/cf-k8s-api/repositories"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"sigs.k8s.io/hierarchical-namespaces/api/v1alpha2"
)

var _ = SuiteDescribe("creating orgs", func(t *testing.T, when spec.G, it spec.S) {
	var orgName string

	it.Before(func() {
		orgName = generateGUID("org")
	})

	it.After(func() {
		deleteOrg(orgName)
	})

	it("creates an org", func() {
		orgsUrl := apiServerRoot + "/v3/organizations"

		body := fmt.Sprintf(`{ "name": "%s" }`, orgName)
		req, err := http.NewRequest(http.MethodPost, orgsUrl, strings.NewReader(body))
		g.Expect(err).NotTo(HaveOccurred())

		resp, err := http.DefaultClient.Do(req)
		g.Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		g.Expect(resp.StatusCode).To(Equal(http.StatusCreated))
		g.Expect(resp.Header["Content-Type"]).To(ConsistOf("application/json"))
		responseMap := map[string]interface{}{}
		g.Expect(json.NewDecoder(resp.Body).Decode(&responseMap)).To(Succeed())
		g.Expect(responseMap["name"]).To(Equal(orgName))

		subnamespaceAnchorList := &v1alpha2.SubnamespaceAnchorList{}
		err = k8sClient.List(context.Background(), subnamespaceAnchorList)
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(subnamespaceAnchorList.Items).To(ContainElement(
			MatchFields(IgnoreExtras, Fields{
				"ObjectMeta": MatchFields(IgnoreExtras, Fields{
					"Labels": HaveKeyWithValue(repositories.OrgNameLabel, orgName),
				}),
			}),
		))
	})
})

var _ = SuiteDescribe("listing orgs", func(t *testing.T, when spec.G, it spec.S) {
	var orgs []hierarchicalNamespace

	it.Before(func() {
		for i := 1; i < 4; i++ {
			orgDetails := createHierarchicalNamespace(rootNamespace, generateGUID("org"+strconv.Itoa(i)), repositories.OrgNameLabel)
			orgs = append(orgs, orgDetails)
			waitForSubnamespaceAnchor(rootNamespace, orgDetails.generatedName)
		}
	})

	it.After(func() {
		for _, org := range orgs {
			deleteOrg(org.generatedName)
		}
	})

	it("returns all 3 orgs", func() {
		g.Eventually(getOrgsFn()).Should(ContainElements(
			MatchFields(IgnoreExtras, Fields{"Name": Equal(orgs[0].label)}),
			MatchFields(IgnoreExtras, Fields{"Name": Equal(orgs[1].label)}),
			MatchFields(IgnoreExtras, Fields{"Name": Equal(orgs[2].label)}),
		))
	})

	when("org names are filtered", func() {
		it("returns orgs 1 & 3", func() {
			g.Eventually(getOrgsFn(orgs[0].label, orgs[2].label)).Should(ContainElements(
				MatchFields(IgnoreExtras, Fields{"Name": Equal(orgs[0].label)}),
				MatchFields(IgnoreExtras, Fields{"Name": Equal(orgs[2].label)}),
			))
			g.Consistently(getOrgsFn(orgs[0].label, orgs[2].label), "2s").ShouldNot(ContainElement(
				MatchFields(IgnoreExtras, Fields{"Name": Equal(orgs[1].label)}),
			))
		})
	})
})

func getOrgsFn(names ...string) func() ([]presenter.OrgResponse, error) {
	return func() ([]presenter.OrgResponse, error) {
		orgsUrl := apiServerRoot + "/v3/organizations"

		if len(names) > 0 {
			orgsUrl += "?names=" + strings.Join(names, ",")
		}

		req, err := http.NewRequest(http.MethodGet, orgsUrl, nil)
		if err != nil {
			return nil, err
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
