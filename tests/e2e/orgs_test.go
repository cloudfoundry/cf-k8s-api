//go:build e2e
// +build e2e

package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/gomega/gstruct"

	"code.cloudfoundry.org/cf-k8s-api/presenter"
	"code.cloudfoundry.org/cf-k8s-api/repositories"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	hnsv1alpha2 "sigs.k8s.io/hierarchical-namespaces/api/v1alpha2"
)

type hierarchicalNamespace struct {
	label         string
	generatedName string
	createdAt     string
	uid           string
	children      []hierarchicalNamespace
}

var _ = Describe("Orgs", func() {
	Describe("Listing Orgs", func() {
		var (
			orgs []hierarchicalNamespace
		)
		BeforeEach(func() {
			for i := 1; i < 4; i++ {
				orgDetails := createHierarchicalNamespace(rootNamespace, generateGUID("org"+strconv.Itoa(i)), repositories.OrgNameLabel)
				orgs = append(orgs, orgDetails)
				waitForSubnamespaceAnchor(rootNamespace, orgDetails.generatedName)
			}
		})

		AfterEach(func() {
			for _, org := range orgs {
				deleteSubnamespace(rootNamespace, org.generatedName)
			}
		})

		It("returns all 3 orgs", func() {
			Eventually(getOrgsFn()).Should(ContainElements(
				MatchFields(IgnoreExtras, Fields{"Name": Equal(orgs[0].label)}),
				MatchFields(IgnoreExtras, Fields{"Name": Equal(orgs[1].label)}),
				MatchFields(IgnoreExtras, Fields{"Name": Equal(orgs[2].label)}),
			))
		})

		When("org names are filtered", func() {
			It("returns orgs 1 & 3", func() {
				Eventually(getOrgsFn(orgs[0].label, orgs[2].label)).Should(ContainElements(
					MatchFields(IgnoreExtras, Fields{"Name": Equal(orgs[0].label)}),
					MatchFields(IgnoreExtras, Fields{"Name": Equal(orgs[2].label)}),
				))
				Consistently(getOrgsFn(orgs[0].label, orgs[2].label), "2s").ShouldNot(ContainElement(
					MatchFields(IgnoreExtras, Fields{"Name": Equal(orgs[1].label)}),
				))
			})
		})
	})

	Describe("listing spaces", func() {
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
					deleteSubnamespace(org.generatedName, space.generatedName)
					waitForNamespaceDeletion(space.generatedName)
				}
				deleteSubnamespace(rootNamespace, org.generatedName)
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
})

func waitForSubnamespaceAnchor(parent, name string) {
	Eventually(func() (bool, error) {
		anchor := &hnsv1alpha2.SubnamespaceAnchor{}
		err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: parent, Name: name}, anchor)
		if err != nil {
			return false, err
		}

		return anchor.Status.State == hnsv1alpha2.Ok, nil
	}, "30s").Should(BeTrue())
}

func waitForNamespaceDeletion(ns string) {
	Eventually(func() (bool, error) {
		err := k8sClient.Get(context.Background(), client.ObjectKey{Name: ns}, &corev1.Namespace{})
		if errors.IsNotFound(err) {
			return true, nil
		}

		fmt.Printf("err = %+v\n", err)

		return false, err
	}, "30s").Should(BeTrue())
}

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

func createHierarchicalNamespace(parentName, cfName, labelKey string) hierarchicalNamespace {
	ctx := context.Background()

	anchor := &hnsv1alpha2.SubnamespaceAnchor{}
	anchor.GenerateName = cfName
	anchor.Namespace = parentName
	anchor.Labels = map[string]string{labelKey: cfName}
	err := k8sClient.Create(ctx, anchor)
	Expect(err).NotTo(HaveOccurred())

	return hierarchicalNamespace{
		label:         cfName,
		generatedName: anchor.Name,
		uid:           string(anchor.UID),
		createdAt:     anchor.CreationTimestamp.Time.UTC().Format(time.RFC3339),
	}
}

func deleteSubnamespace(parent, name string) {
	ctx := context.Background()

	anchor := hnsv1alpha2.SubnamespaceAnchor{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: parent,
			Name:      name,
		},
	}
	err := k8sClient.Delete(ctx, &anchor)
	Expect(err).NotTo(HaveOccurred())
}
