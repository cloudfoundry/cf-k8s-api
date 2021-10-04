package repositories_test

import (
	"context"
	"time"

	"code.cloudfoundry.org/cf-k8s-api/repositories"
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
		orgRepo       *repositories.OrgRepo
		ctx           context.Context
		rootNamespace string

		org1Ns, org2Ns, org3Ns *hnsv1alpha2.SubnamespaceAnchor
	)

	BeforeEach(func() {
		rootNamespace = generateGUID()
		Expect(k8sClient.Create(context.Background(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: rootNamespace}})).To(Succeed())

		orgRepo = repositories.NewOrgRepo(rootNamespace, k8sClient)

		ctx = context.Background()
	})

	It("creates a subnamespace anchor in the root namespace", func() {
		org, err := orgRepo.CreateOrg(ctx, repositories.OrgRecord{
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
		Expect(org.GUID).To(Equal(string(anchorList.Items[0].UID)))
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

	Describe("List", func() {
		var (
			rootNamespace string
			org1Anchor    *hnsv1alpha2.SubnamespaceAnchor
			org2Anchor    *hnsv1alpha2.SubnamespaceAnchor
			org3Anchor    *hnsv1alpha2.SubnamespaceAnchor
		)

		createOrgAnchor := func(name string) *hnsv1alpha2.SubnamespaceAnchor {
			org := &hnsv1alpha2.SubnamespaceAnchor{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: name,
					Namespace:    rootNamespace,
					Labels:       map[string]string{repositories.OrgNameLabel: name},
				},
			}

			Expect(k8sClient.Create(ctx, org)).To(Succeed())
			Expect(k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: org.Name}})).To(Succeed())

			return org
		}

		BeforeEach(func() {
			rootNamespace = generateGUID()
			Expect(k8sClient.Create(context.Background(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: rootNamespace}})).To(Succeed())

			orgRepo = repositories.NewOrgRepo(rootNamespace, k8sClient)

			ctx = context.Background()

			org1Anchor = createOrgAnchor("org1")
			org2Anchor = createOrgAnchor("org2")
			org3Anchor = createOrgAnchor("org3")

		})

		Describe("ListOrgs", func() {
			var (
				orgRepo *repositories.OrgRepo
				ctx     context.Context
			)

			It("returns the 3 orgs", func() {
				orgs, err := orgRepo.FetchOrgs(ctx, nil)
				Expect(err).NotTo(HaveOccurred())
				Expect(orgs).To(ConsistOf(
					repositories.OrgRecord{
						Name:      "org1",
						CreatedAt: org1Anchor.CreationTimestamp.Time,
						UpdatedAt: org1Anchor.CreationTimestamp.Time,
						GUID:      string(org1Anchor.UID),
					},
					repositories.OrgRecord{
						Name:      "org2",
						CreatedAt: org2Ns.CreationTimestamp.Time,
						UpdatedAt: org2Ns.CreationTimestamp.Time,
						GUID:      string(org2Ns.UID),
					},
					repositories.OrgRecord{
						Name:      "org3",
						CreatedAt: org3Anchor.CreationTimestamp.Time,
						UpdatedAt: org3Anchor.CreationTimestamp.Time,
						GUID:      string(org3Anchor.UID),
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
		})

		Describe("List Spaces", func() {
			var (
				space11Anchor *hnsv1alpha2.SubnamespaceAnchor
				space12Anchor *hnsv1alpha2.SubnamespaceAnchor

				space21Anchor *hnsv1alpha2.SubnamespaceAnchor
				space22Anchor *hnsv1alpha2.SubnamespaceAnchor

				space31Anchor *hnsv1alpha2.SubnamespaceAnchor
				space32Anchor *hnsv1alpha2.SubnamespaceAnchor
			)

			createSpaceAnchor := func(name, orgName string) *hnsv1alpha2.SubnamespaceAnchor {
				space := &hnsv1alpha2.SubnamespaceAnchor{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: name,
						Namespace:    orgName,
						Labels:       map[string]string{repositories.SpaceNameLabel: name},
					},
				}

				Expect(k8sClient.Create(ctx, space)).To(Succeed())

				return space
			}

			BeforeEach(func() {
				space11Anchor = createSpaceAnchor("space1", org1Anchor.Name)
				space12Anchor = createSpaceAnchor("space2", org1Anchor.Name)

				space21Anchor = createSpaceAnchor("space1", org2Anchor.Name)
				space22Anchor = createSpaceAnchor("space3", org2Anchor.Name)

				space31Anchor = createSpaceAnchor("space1", org3Anchor.Name)
				space32Anchor = createSpaceAnchor("space4", org3Anchor.Name)
			})

			It("returns the 6 spaces", func() {
				spaces, err := orgRepo.FetchSpaces(ctx, []string{}, []string{})
				Expect(err).NotTo(HaveOccurred())

				Expect(spaces).To(ConsistOf(
					repositories.SpaceRecord{
						Name:             "space11",
						CreatedAt:        space11Anchor.CreationTimestamp.Time,
						UpdatedAt:        space11Anchor.CreationTimestamp.Time,
						GUID:             string(space11Anchor.UID),
						OrganizationGUID: string(org1Ns.UID),
					},
					repositories.SpaceRecord{
						Name:             "space12",
						CreatedAt:        space12Anchor.CreationTimestamp.Time,
						UpdatedAt:        space12Anchor.CreationTimestamp.Time,
						GUID:             string(space12Anchor.UID),
						OrganizationGUID: string(org1Ns.UID),
					},
					repositories.SpaceRecord{
						Name:             "space21",
						CreatedAt:        space21Anchor.CreationTimestamp.Time,
						UpdatedAt:        space21Anchor.CreationTimestamp.Time,
						GUID:             string(space21Anchor.UID),
						OrganizationGUID: string(org2Ns.UID),
					},
					repositories.SpaceRecord{
						Name:             "space22",
						CreatedAt:        space22Anchor.CreationTimestamp.Time,
						UpdatedAt:        space22Anchor.CreationTimestamp.Time,
						GUID:             string(space22Anchor.UID),
						OrganizationGUID: string(org2Ns.UID),
					},
					repositories.SpaceRecord{
						Name:             "space31",
						CreatedAt:        space31Anchor.CreationTimestamp.Time,
						UpdatedAt:        space31Anchor.CreationTimestamp.Time,
						GUID:             string(space31Anchor.UID),
						OrganizationGUID: string(org3Ns.UID),
					},
					repositories.SpaceRecord{
						Name:             "space32",
						CreatedAt:        space32Anchor.CreationTimestamp.Time,
						UpdatedAt:        space32Anchor.CreationTimestamp.Time,
						GUID:             string(space32Anchor.UID),
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
})
