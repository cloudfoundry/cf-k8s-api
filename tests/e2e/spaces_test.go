//go:build e2e
// +build e2e

package e2e_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	"code.cloudfoundry.org/cf-k8s-api/repositories"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Spaces", func() {
	var orgs []hierarchicalNamespace

	BeforeEach(func() {
		for i := 1; i <= 3; i++ {
			orgDetails := createHierarchicalNamespace(rootNamespace, generateGUID("org"+strconv.Itoa(i)), repositories.OrgNameLabel)
			waitForSubnamespaceAnchor(rootNamespace, orgDetails.generatedName)

			for j := 1; j <= 2; j++ {
				spaceDetails := createHierarchicalNamespace(orgDetails.generatedName, generateGUID("space"+strconv.Itoa(j)), repositories.SpaceNameLabel)
				waitForSubnamespaceAnchor(orgDetails.generatedName, spaceDetails.generatedName)
				orgDetails.children = append(orgDetails.children, spaceDetails)
			}

			orgs = append(orgs, orgDetails)
		}
	})

	AfterEach(func() {
		for _, org := range orgs {
			for _, space := range org.children {
				deleteSubnamespaceAnchor(org.generatedName, space.generatedName)
				waitForNamespaceDeletion(space.generatedName)
			}
			deleteOrg(org.generatedName)
		}
	})

	It("lists all the spaces", func() {
		responseBody, err := getSpaces()
		Expect(err).NotTo(HaveOccurred())

		response := map[string]interface{}{}
		Expect(json.Unmarshal([]byte(responseBody), &response)).To(Succeed())

		pagination, ok := response["pagination"].(map[string]interface{})
		Expect(ok).To(BeTrue())

		Expect(pagination["total_results"]).To(BeNumerically("==", 6))
		Expect(response["resources"]).To(ConsistOf(
			HaveKeyWithValue("name", orgs[0].children[0].label),
			HaveKeyWithValue("name", orgs[0].children[1].label),
			HaveKeyWithValue("name", orgs[1].children[0].label),
			HaveKeyWithValue("name", orgs[1].children[1].label),
			HaveKeyWithValue("name", orgs[2].children[0].label),
			HaveKeyWithValue("name", orgs[2].children[1].label),
		))
	})

	When("filtering by organization GUIDs", func() {
		It("only lists spaces beloging to the orgs", func() {
			respJSON, err := getSpacesWithQuery(map[string]string{"organization_guids": fmt.Sprintf("%s,%s", orgs[0].uid, orgs[2].uid)})
			Expect(err).NotTo(HaveOccurred())
			var resp map[string]interface{}
			Expect(json.Unmarshal([]byte(respJSON), &resp)).To(Succeed())

			Expect(resp).To(HaveKey("resources"))
			Expect(resp["resources"]).To(ConsistOf(
				HaveKeyWithValue("name", orgs[0].children[0].label),
				HaveKeyWithValue("name", orgs[0].children[1].label),
				HaveKeyWithValue("name", orgs[2].children[0].label),
				HaveKeyWithValue("name", orgs[2].children[1].label),
			))
		})
	})
})

func getSpaces() (string, error) {
	return getSpacesWithQuery(nil)
}

func getSpacesWithQuery(query map[string]string) (string, error) {
	spacesUrl, err := url.Parse(apiServerRoot)
	if err != nil {
		return "", err
	}
	spacesUrl.Path = "/v3/spaces"
	values := url.Values{}
	for key, val := range query {
		values.Set(key, val)
	}
	spacesUrl.RawQuery = values.Encode()

	resp, err := http.Get(spacesUrl.String())
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(bodyBytes), nil
}
