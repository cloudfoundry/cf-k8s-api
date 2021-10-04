package repositories_test

import (
	"context"
	"testing"
	"time"

	"code.cloudfoundry.org/cf-k8s-api/repositories"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/sclevine/spec"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
	hnsv1alpha2 "sigs.k8s.io/hierarchical-namespaces/api/v1alpha2"
)

var (
	_ = SuiteDescribe("Org Repo List", testList)
	_ = SuiteDescribe("Org Repo Create Org", testCreateOrg)
)

func testCreateOrg(t *testing.T, when spec.G, it spec.S) {
	g := NewWithT(t)

	var (
		orgRepo       *repositories.OrgRepo
		ctx           context.Context
		rootNamespace string
	)

	it.Before(func() {
		rootNamespace = generateGUID()
		g.Expect(k8sClient.Create(context.Background(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: rootNamespace}})).To(Succeed())

		orgRepo = repositories.NewOrgRepo(rootNamespace, k8sClient)

		ctx = context.Background()
	})

	it("creates a subnamespace anchor in the root namespace", func() {
		org, err := orgRepo.CreateOrg(ctx, repositories.OrgRecord{
			Name: "our-org",
		})
		g.Expect(err).NotTo(HaveOccurred())

		namesRequirement, err := labels.NewRequirement(repositories.OrgNameLabel, selection.Equals, []string{"our-org"})
		g.Expect(err).NotTo(HaveOccurred())
		anchorList := hnsv1alpha2.SubnamespaceAnchorList{}
		err = k8sClient.List(ctx, &anchorList, client.InNamespace(rootNamespace), client.MatchingLabelsSelector{
			Selector: labels.NewSelector().Add(*namesRequirement),
		})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(anchorList.Items).To(HaveLen(1))

		g.Expect(org.Name).To(Equal("our-org"))
		g.Expect(org.GUID).To(Equal(string(anchorList.Items[0].UID)))
		g.Expect(org.CreatedAt).To(BeTemporally("~", time.Now(), time.Second))
		g.Expect(org.UpdatedAt).To(BeTemporally("~", time.Now(), time.Second))
	})

	when("the client fails to create the org", func() {
		it("returns an error", func() {
			_, err := orgRepo.CreateOrg(ctx, repositories.OrgRecord{
				Name: "this-string-has-illegal-characters-ц",
			})
			g.Expect(err).To(HaveOccurred())
		})
	})
}

func testList(t *testing.T, when spec.G, it spec.S) {
	g := NewWithT(t)

	var (
		orgRepo       *repositories.OrgRepo
		ctx           context.Context
		rootNamespace string

		org1Anchor, org2Anchor, org3Anchor                                                       *hnsv1alpha2.SubnamespaceAnchor
		space11Anchor, space12Anchor, space21Anchor, space22Anchor, space31Anchor, space32Anchor *hnsv1alpha2.SubnamespaceAnchor
	)

	createOrgAnchor := func(name string) *hnsv1alpha2.SubnamespaceAnchor {
		org := &hnsv1alpha2.SubnamespaceAnchor{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: name,
				Namespace:    rootNamespace,
				Labels:       map[string]string{repositories.OrgNameLabel: name},
			},
		}

		g.Expect(k8sClient.Create(ctx, org)).To(Succeed())
		g.Expect(k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: org.Name}})).To(Succeed())

		return org
	}

	createSpaceAnchor := func(name, orgName string) *hnsv1alpha2.SubnamespaceAnchor {
		space := &hnsv1alpha2.SubnamespaceAnchor{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: name,
				Namespace:    orgName,
				Labels:       map[string]string{repositories.SpaceNameLabel: name},
			},
		}

		g.Expect(k8sClient.Create(ctx, space)).To(Succeed())

		return space
	}

	it.Before(func() {
		rootNamespace = generateGUID()
		g.Expect(k8sClient.Create(context.Background(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: rootNamespace}})).To(Succeed())

		orgRepo = repositories.NewOrgRepo(rootNamespace, k8sClient)

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

	it("returns the 3 orgs", func() {
		orgs, err := orgRepo.FetchOrgs(ctx, nil)
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(orgs).To(ConsistOf(
			repositories.OrgRecord{
				Name:      "org1",
				CreatedAt: org1Anchor.CreationTimestamp.Time,
				UpdatedAt: org1Anchor.CreationTimestamp.Time,
				GUID:      string(org1Anchor.UID),
			},
			repositories.OrgRecord{
				Name:      "org2",
				CreatedAt: org2Anchor.CreationTimestamp.Time,
				UpdatedAt: org2Anchor.CreationTimestamp.Time,
				GUID:      string(org2Anchor.UID),
			},
			repositories.OrgRecord{
				Name:      "org3",
				CreatedAt: org3Anchor.CreationTimestamp.Time,
				UpdatedAt: org3Anchor.CreationTimestamp.Time,
				GUID:      string(org3Anchor.UID),
			},
		))
	})

	when("we filter for org1 and org3", func() {
		it("returns just those", func() {
			orgs, err := orgRepo.FetchOrgs(ctx, []string{"org1", "org3"})
			g.Expect(err).NotTo(HaveOccurred())

			g.Expect(orgs).To(ConsistOf(
				repositories.OrgRecord{
					Name:      "org1",
					CreatedAt: org1Anchor.CreationTimestamp.Time,
					UpdatedAt: org1Anchor.CreationTimestamp.Time,
					GUID:      string(org1Anchor.UID),
				},
				repositories.OrgRecord{
					Name:      "org3",
					CreatedAt: org3Anchor.CreationTimestamp.Time,
					UpdatedAt: org3Anchor.CreationTimestamp.Time,
					GUID:      string(org3Anchor.UID),
				},
			))
		})
	})

	it("returns the 6 spaces", func() {
		spaces, err := orgRepo.FetchSpaces(ctx, []string{}, []string{})
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(spaces).To(ConsistOf(
			repositories.SpaceRecord{
				Name:             "space1",
				CreatedAt:        space11Anchor.CreationTimestamp.Time,
				UpdatedAt:        space11Anchor.CreationTimestamp.Time,
				GUID:             string(space11Anchor.UID),
				OrganizationGUID: string(org1Anchor.UID),
			},
			repositories.SpaceRecord{
				Name:             "space2",
				CreatedAt:        space12Anchor.CreationTimestamp.Time,
				UpdatedAt:        space12Anchor.CreationTimestamp.Time,
				GUID:             string(space12Anchor.UID),
				OrganizationGUID: string(org1Anchor.UID),
			},
			repositories.SpaceRecord{
				Name:             "space1",
				CreatedAt:        space21Anchor.CreationTimestamp.Time,
				UpdatedAt:        space21Anchor.CreationTimestamp.Time,
				GUID:             string(space21Anchor.UID),
				OrganizationGUID: string(org2Anchor.UID),
			},
			repositories.SpaceRecord{
				Name:             "space3",
				CreatedAt:        space22Anchor.CreationTimestamp.Time,
				UpdatedAt:        space22Anchor.CreationTimestamp.Time,
				GUID:             string(space22Anchor.UID),
				OrganizationGUID: string(org2Anchor.UID),
			},
			repositories.SpaceRecord{
				Name:             "space1",
				CreatedAt:        space31Anchor.CreationTimestamp.Time,
				UpdatedAt:        space31Anchor.CreationTimestamp.Time,
				GUID:             string(space31Anchor.UID),
				OrganizationGUID: string(org3Anchor.UID),
			},
			repositories.SpaceRecord{
				Name:             "space4",
				CreatedAt:        space32Anchor.CreationTimestamp.Time,
				UpdatedAt:        space32Anchor.CreationTimestamp.Time,
				GUID:             string(space32Anchor.UID),
				OrganizationGUID: string(org3Anchor.UID),
			},
		))
	})

	when("filtering by org guids", func() {
		it("only retruns the spaces belonging to the specified org guids", func() {
			spaces, err := orgRepo.FetchSpaces(ctx, []string{string(org1Anchor.UID), string(org3Anchor.UID), "does-not-exist"}, []string{})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(spaces).To(ConsistOf(
				MatchFields(IgnoreExtras, Fields{"Name": Equal("space1"), "OrganizationGUID": Equal(string(org1Anchor.UID))}),
				MatchFields(IgnoreExtras, Fields{"Name": Equal("space1"), "OrganizationGUID": Equal(string(org3Anchor.UID))}),
				MatchFields(IgnoreExtras, Fields{"Name": Equal("space2")}),
				MatchFields(IgnoreExtras, Fields{"Name": Equal("space4")}),
			))
		})
	})

	when("filtering by space names", func() {
		it("only retruns the spaces matching the specified names", func() {
			spaces, err := orgRepo.FetchSpaces(ctx, []string{}, []string{"space1", "space3", "does-not-exist"})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(spaces).To(ConsistOf(
				MatchFields(IgnoreExtras, Fields{"Name": Equal("space1"), "OrganizationGUID": Equal(string(org1Anchor.UID))}),
				MatchFields(IgnoreExtras, Fields{"Name": Equal("space1"), "OrganizationGUID": Equal(string(org2Anchor.UID))}),
				MatchFields(IgnoreExtras, Fields{"Name": Equal("space1"), "OrganizationGUID": Equal(string(org3Anchor.UID))}),
				MatchFields(IgnoreExtras, Fields{"Name": Equal("space3")}),
			))
		})
	})

	when("filtering by org guids and space names", func() {
		it("only retruns the spaces matching the specified names", func() {
			spaces, err := orgRepo.FetchSpaces(ctx, []string{string(org1Anchor.UID), string(org2Anchor.UID)}, []string{"space1", "space2", "space4"})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(spaces).To(ConsistOf(
				MatchFields(IgnoreExtras, Fields{"Name": Equal("space1"), "OrganizationGUID": Equal(string(org1Anchor.UID))}),
				MatchFields(IgnoreExtras, Fields{"Name": Equal("space1"), "OrganizationGUID": Equal(string(org2Anchor.UID))}),
				MatchFields(IgnoreExtras, Fields{"Name": Equal("space2")}),
			))
		})
	})

	when("filtering by space names that don't exist", func() {
		it("only retruns the spaces matching the specified names", func() {
			spaces, err := orgRepo.FetchSpaces(ctx, []string{}, []string{"does-not-exist", "still-does-not-exist"})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(spaces).To(BeEmpty())
		})
	})

	when("filtering by org uids that don't exist", func() {
		it("only retruns the spaces matching the specified names", func() {
			spaces, err := orgRepo.FetchSpaces(ctx, []string{"does-not-exist", "still-does-not-exist"}, []string{})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(spaces).To(BeEmpty())
		})
	})
}
