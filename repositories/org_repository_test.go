package repositories_test

import (
	"context"
	"time"

	"code.cloudfoundry.org/cf-k8s-api/repositories"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
	hnsv1alpha2 "sigs.k8s.io/hierarchical-namespaces/api/v1alpha2"
)

var _ = Describe("OrgRepository", func() {
	var (
		rootNamespace string
		orgRepo       *repositories.OrgRepo
	)

	BeforeEach(func() {
		rootNamespace = generateGUID()
		Expect(k8sClient.Create(context.Background(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: rootNamespace}})).To(Succeed())
		orgRepo = repositories.NewOrgRepo(rootNamespace, k8sClient)
	})

	Describe("Create Org", func() {
		var ctx context.Context

		BeforeEach(func() {
			ctx = context.Background()
		})

		It("creates a subnamespace anchor in the root namespace", func() {
			org, err := orgRepo.CreateOrg(ctx, repositories.OrgRecord{
				GUID: "some-guid",
				Name: "our-org",
			})
			Expect(err).NotTo(HaveOccurred())

			namesRequirement, err := labels.NewRequirement(repositories.OrgNameLabel, selection.Equals, []string{"our-org"})
			Expect(err).NotTo(HaveOccurred())
			anchorList := hnsv1alpha2.SubnamespaceAnchorList{}
			err = k8sClient.List(ctx, &anchorList, client.InNamespace(rootNamespace), client.MatchingLabelsSelector{
				Selector: labels.NewSelector().Add(*namesRequirement),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(anchorList.Items).To(HaveLen(1))

			Expect(org.Name).To(Equal("our-org"))
			Expect(org.GUID).To(Equal("some-guid"))
			Expect(org.CreatedAt).To(BeTemporally("~", time.Now(), time.Second))
			Expect(org.UpdatedAt).To(BeTemporally("~", time.Now(), time.Second))
		})

		When("the client fails to create the org", func() {
			It("returns an error", func() {
				_, err := orgRepo.CreateOrg(ctx, repositories.OrgRecord{
					Name: "this-string-has-illegal-characters-ц",
				})
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("ListOrgs", func() {
		var (
			ctx context.Context

			org1Anchor, org2Anchor, org3Anchor                                                       *hnsv1alpha2.SubnamespaceAnchor
			space11Anchor, space12Anchor, space21Anchor, space22Anchor, space31Anchor, space32Anchor *hnsv1alpha2.SubnamespaceAnchor
		)

		createOrgAnchor := func(name string) *hnsv1alpha2.SubnamespaceAnchor {
			guid := uuid.New().String()
			org := &hnsv1alpha2.SubnamespaceAnchor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      guid,
					Namespace: rootNamespace,
					Labels:    map[string]string{repositories.OrgNameLabel: name},
				},
			}

			Expect(k8sClient.Create(ctx, org)).To(Succeed())
			Expect(k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: org.Name}})).To(Succeed())

			return org
		}

		createSpaceAnchor := func(name, orgName string) *hnsv1alpha2.SubnamespaceAnchor {
			guid := uuid.New().String()
			space := &hnsv1alpha2.SubnamespaceAnchor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      guid,
					Namespace: orgName,
					Labels:    map[string]string{repositories.SpaceNameLabel: name},
				},
			}

			Expect(k8sClient.Create(ctx, space)).To(Succeed())

			return space
		}

		BeforeEach(func() {
			ctx = context.Background()

			org1Anchor = createOrgAnchor("org1")
			org2Anchor = createOrgAnchor("org2")
			org3Anchor = createOrgAnchor("org3")

			space11Anchor = createSpaceAnchor("space1", org1Anchor.Name)
			space12Anchor = createSpaceAnchor("space2", org1Anchor.Name)

			space21Anchor = createSpaceAnchor("space1", org2Anchor.Name)
			space22Anchor = createSpaceAnchor("space3", org2Anchor.Name)

			space31Anchor = createSpaceAnchor("space1", org3Anchor.Name)
			space32Anchor = createSpaceAnchor("space4", org3Anchor.Name)
		})

		It("returns the 3 orgs", func() {
			orgs, err := orgRepo.FetchOrgs(ctx, nil)
			Expect(err).NotTo(HaveOccurred())

			Expect(orgs).To(ConsistOf(
				repositories.OrgRecord{
					Name:      "org1",
					CreatedAt: org1Anchor.CreationTimestamp.Time,
					UpdatedAt: org1Anchor.CreationTimestamp.Time,
					GUID:      org1Anchor.Name,
				},
				repositories.OrgRecord{
					Name:      "org2",
					CreatedAt: org2Anchor.CreationTimestamp.Time,
					UpdatedAt: org2Anchor.CreationTimestamp.Time,
					GUID:      org2Anchor.Name,
				},
				repositories.OrgRecord{
					Name:      "org3",
					CreatedAt: org3Anchor.CreationTimestamp.Time,
					UpdatedAt: org3Anchor.CreationTimestamp.Time,
					GUID:      org3Anchor.Name,
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
						CreatedAt: org1Anchor.CreationTimestamp.Time,
						UpdatedAt: org1Anchor.CreationTimestamp.Time,
						GUID:      org1Anchor.Name,
					},
					repositories.OrgRecord{
						Name:      "org3",
						CreatedAt: org3Anchor.CreationTimestamp.Time,
						UpdatedAt: org3Anchor.CreationTimestamp.Time,
						GUID:      org3Anchor.Name,
					},
				))
			})
		})

		It("returns the 6 spaces", func() {
			spaces, err := orgRepo.FetchSpaces(ctx, []string{}, []string{})
			Expect(err).NotTo(HaveOccurred())

			Expect(spaces).To(ConsistOf(
				repositories.SpaceRecord{
					Name:             "space1",
					CreatedAt:        space11Anchor.CreationTimestamp.Time,
					UpdatedAt:        space11Anchor.CreationTimestamp.Time,
					GUID:             space11Anchor.Name,
					OrganizationGUID: org1Anchor.Name,
				},
				repositories.SpaceRecord{
					Name:             "space2",
					CreatedAt:        space12Anchor.CreationTimestamp.Time,
					UpdatedAt:        space12Anchor.CreationTimestamp.Time,
					GUID:             space12Anchor.Name,
					OrganizationGUID: org1Anchor.Name,
				},
				repositories.SpaceRecord{
					Name:             "space1",
					CreatedAt:        space21Anchor.CreationTimestamp.Time,
					UpdatedAt:        space21Anchor.CreationTimestamp.Time,
					GUID:             space21Anchor.Name,
					OrganizationGUID: org2Anchor.Name,
				},
				repositories.SpaceRecord{
					Name:             "space3",
					CreatedAt:        space22Anchor.CreationTimestamp.Time,
					UpdatedAt:        space22Anchor.CreationTimestamp.Time,
					GUID:             space22Anchor.Name,
					OrganizationGUID: org2Anchor.Name,
				},
				repositories.SpaceRecord{
					Name:             "space1",
					CreatedAt:        space31Anchor.CreationTimestamp.Time,
					UpdatedAt:        space31Anchor.CreationTimestamp.Time,
					GUID:             space31Anchor.Name,
					OrganizationGUID: org3Anchor.Name,
				},
				repositories.SpaceRecord{
					Name:             "space4",
					CreatedAt:        space32Anchor.CreationTimestamp.Time,
					UpdatedAt:        space32Anchor.CreationTimestamp.Time,
					GUID:             space32Anchor.Name,
					OrganizationGUID: org3Anchor.Name,
				},
			))
		})

		When("filtering by org guids", func() {
			It("only retruns the spaces belonging to the specified org guids", func() {
				spaces, err := orgRepo.FetchSpaces(ctx, []string{string(org1Anchor.Name), string(org3Anchor.Name), "does-not-exist"}, []string{})
				Expect(err).NotTo(HaveOccurred())
				Expect(spaces).To(ConsistOf(
					MatchFields(IgnoreExtras, Fields{
						"Name":             Equal("space1"),
						"OrganizationGUID": Equal(string(org1Anchor.Name)),
					}),
					MatchFields(IgnoreExtras, Fields{
						"Name":             Equal("space1"),
						"OrganizationGUID": Equal(string(org3Anchor.Name)),
					}),
					MatchFields(IgnoreExtras, Fields{"Name": Equal("space2")}),
					MatchFields(IgnoreExtras, Fields{"Name": Equal("space4")}),
				))
			})
		})

		When("filtering by space names", func() {
			It("only retruns the spaces matching the specified names", func() {
				spaces, err := orgRepo.FetchSpaces(ctx, []string{}, []string{"space1", "space3", "does-not-exist"})
				Expect(err).NotTo(HaveOccurred())
				Expect(spaces).To(ConsistOf(
					MatchFields(IgnoreExtras, Fields{
						"Name":             Equal("space1"),
						"OrganizationGUID": Equal(string(org1Anchor.Name)),
					}),
					MatchFields(IgnoreExtras, Fields{
						"Name":             Equal("space1"),
						"OrganizationGUID": Equal(string(org2Anchor.Name)),
					}),
					MatchFields(IgnoreExtras, Fields{
						"Name":             Equal("space1"),
						"OrganizationGUID": Equal(string(org3Anchor.Name)),
					}),
					MatchFields(IgnoreExtras, Fields{"Name": Equal("space3")}),
				))
			})
		})

		When("filtering by org guids and space names", func() {
			It("only retruns the spaces matching the specified names", func() {
				spaces, err := orgRepo.FetchSpaces(ctx, []string{string(org1Anchor.Name), string(org2Anchor.Name)}, []string{"space1", "space2", "space4"})
				Expect(err).NotTo(HaveOccurred())
				Expect(spaces).To(ConsistOf(
					MatchFields(IgnoreExtras, Fields{
						"Name":             Equal("space1"),
						"OrganizationGUID": Equal(string(org1Anchor.Name)),
					}),
					MatchFields(IgnoreExtras, Fields{
						"Name":             Equal("space1"),
						"OrganizationGUID": Equal(string(org2Anchor.Name)),
					}),
					MatchFields(IgnoreExtras, Fields{"Name": Equal("space2")}),
				))
			})
		})

		When("filtering by space names that don't exist", func() {
			It("only retruns the spaces matching the specified names", func() {
				spaces, err := orgRepo.FetchSpaces(ctx, []string{}, []string{"does-not-exist", "still-does-not-exist"})
				Expect(err).NotTo(HaveOccurred())
				Expect(spaces).To(BeEmpty())
			})
		})

		When("filtering by org uids that don't exist", func() {
			It("only retruns the spaces matching the specified names", func() {
				spaces, err := orgRepo.FetchSpaces(ctx, []string{"does-not-exist", "still-does-not-exist"}, []string{})
				Expect(err).NotTo(HaveOccurred())
				Expect(spaces).To(BeEmpty())
			})
		})
	})
})
