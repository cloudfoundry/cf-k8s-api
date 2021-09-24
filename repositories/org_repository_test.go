package repositories_test

import (
	"context"

	"code.cloudfoundry.org/cf-k8s-api/repositories"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	hnsv1alpha2 "sigs.k8s.io/hierarchical-namespaces/api/v1alpha2"
)

var _ = Describe("OrgRepository", func() {
	Describe("ListOrgs", func() {
		var (
			orgRepo                                                          *repositories.OrgRepo
			ctx                                                              context.Context
			rootNamespace                                                    string
			org1Ns, org2Ns, org3Ns                                           *hnsv1alpha2.SubnamespaceAnchor
			space11Ns, space12Ns, space21Ns, space22Ns, space31Ns, space32Ns *hnsv1alpha2.SubnamespaceAnchor
		)

		BeforeEach(func() {
			rootNamespace = generateGUID()
			Expect(k8sClient.Create(context.Background(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: rootNamespace}})).To(Succeed())

			orgRepo = repositories.NewOrgRepo(rootNamespace, k8sClient)

			ctx = context.Background()

			org1Ns = &hnsv1alpha2.SubnamespaceAnchor{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "org1",
					Namespace:    rootNamespace,
					Labels:       map[string]string{repositories.OrgNameLabel: "org1"},
				},
			}
			org2Ns = &hnsv1alpha2.SubnamespaceAnchor{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "org2",
					Namespace:    rootNamespace,
					Labels:       map[string]string{repositories.OrgNameLabel: "org2"},
				},
			}
			org3Ns = &hnsv1alpha2.SubnamespaceAnchor{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "org3",
					Namespace:    rootNamespace,
					Labels:       map[string]string{repositories.OrgNameLabel: "org3"},
				},
			}
			Expect(k8sClient.Create(ctx, org1Ns)).To(Succeed())
			Expect(k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: org1Ns.Name}})).To(Succeed())
			Expect(k8sClient.Create(ctx, org2Ns)).To(Succeed())
			Expect(k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: org2Ns.Name}})).To(Succeed())
			Expect(k8sClient.Create(ctx, org3Ns)).To(Succeed())
			Expect(k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: org3Ns.Name}})).To(Succeed())

			space11Ns = &hnsv1alpha2.SubnamespaceAnchor{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "space11",
					Namespace:    org1Ns.Name,
					Labels:       map[string]string{repositories.SpaceNameLabel: "space11"},
				},
			}
			space12Ns = &hnsv1alpha2.SubnamespaceAnchor{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "space12",
					Namespace:    org1Ns.Name,
					Labels:       map[string]string{repositories.SpaceNameLabel: "space12"},
				},
			}
			space21Ns = &hnsv1alpha2.SubnamespaceAnchor{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "space21",
					Namespace:    org2Ns.Name,
					Labels:       map[string]string{repositories.SpaceNameLabel: "space21"},
				},
			}
			space22Ns = &hnsv1alpha2.SubnamespaceAnchor{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "space22",
					Namespace:    org2Ns.Name,
					Labels:       map[string]string{repositories.SpaceNameLabel: "space22"},
				},
			}
			space31Ns = &hnsv1alpha2.SubnamespaceAnchor{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "space31",
					Namespace:    org3Ns.Name,
					Labels:       map[string]string{repositories.SpaceNameLabel: "space31"},
				},
			}
			space32Ns = &hnsv1alpha2.SubnamespaceAnchor{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "space32",
					Namespace:    org3Ns.Name,
					Labels:       map[string]string{repositories.SpaceNameLabel: "space32"},
				},
			}

			Expect(k8sClient.Create(ctx, space11Ns)).To(Succeed())
			Expect(k8sClient.Create(ctx, space12Ns)).To(Succeed())
			Expect(k8sClient.Create(ctx, space21Ns)).To(Succeed())
			Expect(k8sClient.Create(ctx, space22Ns)).To(Succeed())
			Expect(k8sClient.Create(ctx, space31Ns)).To(Succeed())
			Expect(k8sClient.Create(ctx, space32Ns)).To(Succeed())
		})

		It("returns the 3 orgs", func() {
			orgs, err := orgRepo.FetchOrgs(ctx, nil)
			Expect(err).NotTo(HaveOccurred())

			Expect(orgs).To(ConsistOf(
				repositories.OrgRecord{
					Name:      "org1",
					CreatedAt: org1Ns.CreationTimestamp.Time,
					UpdatedAt: org1Ns.CreationTimestamp.Time,
					GUID:      string(org1Ns.UID),
				},
				repositories.OrgRecord{
					Name:      "org2",
					CreatedAt: org2Ns.CreationTimestamp.Time,
					UpdatedAt: org2Ns.CreationTimestamp.Time,
					GUID:      string(org2Ns.UID),
				},
				repositories.OrgRecord{
					Name:      "org3",
					CreatedAt: org3Ns.CreationTimestamp.Time,
					UpdatedAt: org3Ns.CreationTimestamp.Time,
					GUID:      string(org3Ns.UID),
				},
			))
		})

		When("we filter for org1 and org3", func() {
			It("returns just those", func() {
				orgs, err := orgRepo.FetchOrgs(ctx, []string{"org1", "org3"})
				Expect(err).NotTo(HaveOccurred())

				Expect(orgs).To(ConsistOf(
					repositories.OrgRecord{
						Name:      "org1",
						CreatedAt: org1Ns.CreationTimestamp.Time,
						UpdatedAt: org1Ns.CreationTimestamp.Time,
						GUID:      string(org1Ns.UID),
					},
					repositories.OrgRecord{
						Name:      "org3",
						CreatedAt: org3Ns.CreationTimestamp.Time,
						UpdatedAt: org3Ns.CreationTimestamp.Time,
						GUID:      string(org3Ns.UID),
					},
				))
			})
		})

		It("returns the 6 spaces", func() {
			spaces, err := orgRepo.FetchSpaces(ctx, []string{}, []string{})
			Expect(err).NotTo(HaveOccurred())

			Expect(spaces).To(ConsistOf(
				repositories.SpaceRecord{
					Name:             "space11",
					CreatedAt:        space11Ns.CreationTimestamp.Time,
					UpdatedAt:        space11Ns.CreationTimestamp.Time,
					GUID:             string(space11Ns.UID),
					OrganizationGUID: string(org1Ns.UID),
				},
				repositories.SpaceRecord{
					Name:             "space12",
					CreatedAt:        space12Ns.CreationTimestamp.Time,
					UpdatedAt:        space12Ns.CreationTimestamp.Time,
					GUID:             string(space12Ns.UID),
					OrganizationGUID: string(org1Ns.UID),
				},
				repositories.SpaceRecord{
					Name:             "space21",
					CreatedAt:        space21Ns.CreationTimestamp.Time,
					UpdatedAt:        space21Ns.CreationTimestamp.Time,
					GUID:             string(space21Ns.UID),
					OrganizationGUID: string(org2Ns.UID),
				},
				repositories.SpaceRecord{
					Name:             "space22",
					CreatedAt:        space22Ns.CreationTimestamp.Time,
					UpdatedAt:        space22Ns.CreationTimestamp.Time,
					GUID:             string(space22Ns.UID),
					OrganizationGUID: string(org2Ns.UID),
				},
				repositories.SpaceRecord{
					Name:             "space31",
					CreatedAt:        space31Ns.CreationTimestamp.Time,
					UpdatedAt:        space31Ns.CreationTimestamp.Time,
					GUID:             string(space31Ns.UID),
					OrganizationGUID: string(org3Ns.UID),
				},
				repositories.SpaceRecord{
					Name:             "space32",
					CreatedAt:        space32Ns.CreationTimestamp.Time,
					UpdatedAt:        space32Ns.CreationTimestamp.Time,
					GUID:             string(space32Ns.UID),
					OrganizationGUID: string(org3Ns.UID),
				},
			))
		})

		When("filtering by org guids", func() {
			It("only retruns the spaces belonging to the specified org guids", func() {
				spaces, err := orgRepo.FetchSpaces(ctx, []string{string(org1Ns.UID), string(org3Ns.UID)}, []string{})
				Expect(err).NotTo(HaveOccurred())
				Expect(spaces).To(ConsistOf(
					MatchFields(IgnoreExtras, Fields{"Name": Equal("space11")}),
					MatchFields(IgnoreExtras, Fields{"Name": Equal("space12")}),
					MatchFields(IgnoreExtras, Fields{"Name": Equal("space31")}),
					MatchFields(IgnoreExtras, Fields{"Name": Equal("space32")}),
				))
			})
		})

		When("filtering by space names", func() {
			It("only retruns the spaces matching the specified names", func() {
				spaces, err := orgRepo.FetchSpaces(ctx, []string{}, []string{"space11", "space31", "space41"})
				Expect(err).NotTo(HaveOccurred())
				Expect(spaces).To(ConsistOf(
					MatchFields(IgnoreExtras, Fields{"Name": Equal("space11")}),
					MatchFields(IgnoreExtras, Fields{"Name": Equal("space31")}),
				))
			})
		})
	})
})
