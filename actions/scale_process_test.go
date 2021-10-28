package actions_test

import (
	"context"

	. "code.cloudfoundry.org/cf-k8s-api/actions"
	"code.cloudfoundry.org/cf-k8s-api/actions/fake"
	"code.cloudfoundry.org/cf-k8s-api/repositories"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("ScaleProcessAction", func() {
	const (
		testProcessGUID = "test-process-guid"
		testProcessSpaceGUID = "test-namespace"

		initialInstances = 1
		initialMemoryMB = 256
		intialDiskQuotaMB = 1024
	)
	var (
		processRepo   *fake.CFProcessRepository

		updatedProcessRecord *repositories.ProcessRecord

		scaleProcessAction *ScaleProcess

		testClient *fake.Client
		testScale *repositories.ProcessScale

		responseRecord repositories.ProcessRecord
		responseErr error
	)

	BeforeEach(func() {
		processRepo = new(fake.CFProcessRepository)
		testClient = new(fake.Client)

		processRepo.FetchProcessReturns(repositories.ProcessRecord{
			GUID:        testProcessGUID,
			SpaceGUID:   testProcessSpaceGUID,
			Instances:   initialInstances,
			MemoryMB:    initialMemoryMB,
			DiskQuotaMB: intialDiskQuotaMB,
		}, nil)

		updatedProcessRecord = &repositories.ProcessRecord{
			GUID:        testProcessGUID,
			SpaceGUID:   testProcessSpaceGUID,
			AppGUID:     "some-app-guid",
			Type:        "web",
			Command:     "some-command",
			Instances:   initialInstances,
			MemoryMB:    initialMemoryMB,
			DiskQuotaMB: intialDiskQuotaMB,
			Ports:       []int32{8080},
			HealthCheck: repositories.HealthCheck{
				Type: "port",
				Data: repositories.HealthCheckData{
				},
			},
			Labels:      nil,
			Annotations: nil,
			CreatedAt:   "1906-04-18T13:12:00Z",
			UpdatedAt:   "1906-04-18T13:12:01Z",
		}
		processRepo.ScaleProcessReturns(*updatedProcessRecord, nil)

		testScale = &repositories.ProcessScale{
			Instances: nil,
			MemoryMB:  nil,
			DiskMB:    nil,
		}

		scaleProcessAction = NewScaleProcess(processRepo)
	})

	JustBeforeEach(func() {
		responseRecord, responseErr = scaleProcessAction.Invoke(context.Background(), testClient, testProcessGUID, *testScale)
	})

	When("on the happy path", func() {
		It("fetches the process associated with the GUID", func() {

		})
		It("fabricates a ProcessScaleMessage using the inputs and the process GUID and looked-up space", func() {

		})
		It("transparently returns a record from repositories.ProcessScale", func() {

		})
	})
})