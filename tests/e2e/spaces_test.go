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
	"testing"

	"code.cloudfoundry.org/cf-k8s-api/repositories"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
)

var _ = SuiteDescribe("listing spaces", func(t *testing.T, when spec.G, it spec.S) {
	var orgs []hierarchicalNamespace

	it.Before(func() {
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

	it.After(func() {
		for _, org := range orgs {
			for _, space := range org.children {
				deleteSubnamespace(org.generatedName, space.generatedName)
				waitForNamespaceDeletion(space.generatedName)
			}
			deleteSubnamespace(rootNamespace, org.generatedName)
		}
	})

	it("lists all the spaces", func() {
		responseBody, err := getSpaces()
		g.Expect(err).NotTo(HaveOccurred())

		response := map[string]interface{}{}
		g.Expect(json.Unmarshal([]byte(responseBody), &response)).To(Succeed())

		pagination, ok := response["pagination"].(map[string]interface{})
		g.Expect(ok).To(BeTrue())

		g.Expect(pagination["total_results"]).To(BeNumerically("==", 6))
		g.Expect(response["resources"]).To(ConsistOf(
			HaveKeyWithValue("name", orgs[0].children[0].label),
			HaveKeyWithValue("name", orgs[0].children[1].label),
			HaveKeyWithValue("name", orgs[1].children[0].label),
			HaveKeyWithValue("name", orgs[1].children[1].label),
			HaveKeyWithValue("name", orgs[2].children[0].label),
			HaveKeyWithValue("name", orgs[2].children[1].label),
		))
	})

	when("filtering by organization GUIDs", func() {
		it("only lists spaces beloging to the orgs", func() {
			respJSON, err := getSpacesWithQuery(map[string]string{"organization_guids": fmt.Sprintf("%s,%s", orgs[0].uid, orgs[2].uid)})
			g.Expect(err).NotTo(HaveOccurred())
			var resp map[string]interface{}
			g.Expect(json.Unmarshal([]byte(respJSON), &resp)).To(Succeed())

			g.Expect(resp).To(HaveKey("resources"))
			g.Expect(resp["resources"]).To(ConsistOf(
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
