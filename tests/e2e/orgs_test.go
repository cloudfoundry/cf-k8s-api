//go:build e2e
// +build e2e

package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	. "github.com/onsi/gomega/gstruct"

	"code.cloudfoundry.org/cf-k8s-api/presenter"
	"code.cloudfoundry.org/cf-k8s-api/repositories"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	hnsv1alpha2 "sigs.k8s.io/hierarchical-namespaces/api/v1alpha2"
)

var _ = Describe("Orgs", func() {
	Describe("Listing Orgs", func() {
		var org1, org2, org3 string

		BeforeEach(func() {
			org1 = generateGUID()
			org2 = generateGUID()
			org3 = generateGUID()
			createSubnamespaces(rootNamespace, org1, org2, org3)
		})

		AfterEach(func() {
			deleteSubnamespaces(rootNamespace, org1, org2, org3)
		})

		It("returns all 3 orgs", func() {
			Eventually(getOrgs()).Should(ContainElements(
				MatchFields(IgnoreExtras, Fields{"Name": Equal(org1)}),
				MatchFields(IgnoreExtras, Fields{"Name": Equal(org2)}),
				MatchFields(IgnoreExtras, Fields{"Name": Equal(org3)}),
			))
		})

		When("org names are filtered", func() {
			It("returns orgs 1 & 3", func() {
				Eventually(getOrgs(org1, org3)).Should(ContainElements(
					MatchFields(IgnoreExtras, Fields{"Name": Equal(org1)}),
					MatchFields(IgnoreExtras, Fields{"Name": Equal(org3)}),
				))
				Consistently(getOrgs(org1, org3), "2s").ShouldNot(ContainElement(
					MatchFields(IgnoreExtras, Fields{"Name": Equal(org2)}),
				))
			})
		})
	})
})

func getOrgs(names ...string) func() ([]presenter.OrgResponse, error) {
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

func createSubnamespaces(parent string, names ...string) {
	ctx := context.Background()

	for _, name := range names {
		anchor := &hnsv1alpha2.SubnamespaceAnchor{}
		anchor.GenerateName = name
		anchor.Namespace = parent
		anchor.Labels = map[string]string{repositories.OrgNameLabel: name}
		err := k8sClient.Create(ctx, anchor)
		Expect(err).NotTo(HaveOccurred())
	}
}

func deleteSubnamespaces(parent string, names ...string) {
	ctx := context.Background()

	namesRequirement, err := labels.NewRequirement(repositories.OrgNameLabel, selection.In, names)
	Expect(err).NotTo(HaveOccurred())
	namesSelector := client.MatchingLabelsSelector{
		Selector: labels.NewSelector().Add(*namesRequirement),
	}

	err = k8sClient.DeleteAllOf(ctx, &hnsv1alpha2.SubnamespaceAnchor{}, client.InNamespace(parent), namesSelector)
	Expect(err).NotTo(HaveOccurred())
}
