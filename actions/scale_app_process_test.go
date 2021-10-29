package actions_test

import (
	"context"
	"errors"

	. "code.cloudfoundry.org/cf-k8s-api/actions"
	"code.cloudfoundry.org/cf-k8s-api/actions/fake"
	"code.cloudfoundry.org/cf-k8s-api/repositories"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ScaleAppProcessAction", func() {
	const (
		testAppGUID          = "test-app-guid"
		testProcessGUID      = "test-process-guid"
		testProcessSpaceGUID = "test-namespace"

		initialInstances   = 1
		initialMemoryMB    = 256
		initialDiskQuotaMB = 1024
	)
	var (
		appRepo     *fake.CFAppRepository
		processRepo *fake.CFProcessRepository
		processType string

		updatedProcessRecord *repositories.ProcessRecord

		scaleAppProcessAction *ScaleAppProcess

		testClient *fake.Client
		testScale  *repositories.ProcessScale

		responseRecord repositories.ProcessRecord
		responseErr    error
	)

	BeforeEach(func() {
		appRepo = new(fake.CFAppRepository)
		processRepo = new(fake.CFProcessRepository)
		processType = "web"
		testClient = new(fake.Client)

		processRecord := repositories.ProcessRecord{
			GUID:        testProcessGUID,
			SpaceGUID:   testProcessSpaceGUID,
			Instances:   initialInstances,
			MemoryMB:    initialMemoryMB,
			DiskQuotaMB: initialDiskQuotaMB,
		}

		appRepo.FetchAppReturns(repositories.AppRecord{
			Name:      testAppGUID,
			GUID:      testAppGUID,
			SpaceGUID: testProcessSpaceGUID,
		}, nil)

		processRepo.FetchProcessesForAppReturns([]repositories.ProcessRecord{processRecord}, nil)

		processRepo.FetchProcessReturns(processRecord, nil)

		updatedProcessRecord = &repositories.ProcessRecord{
			GUID:        testProcessGUID,
			SpaceGUID:   testProcessSpaceGUID,
			AppGUID:     testAppGUID,
			Type:        processType,
			Command:     "some-command",
			Instances:   initialInstances,
			MemoryMB:    initialMemoryMB,
			DiskQuotaMB: initialDiskQuotaMB,
			Ports:       []int32{8080},
			HealthCheck: repositories.HealthCheck{
				Type: "port",
				Data: repositories.HealthCheckData{},
			},
			Labels:      nil,
			Annotations: nil,
			CreatedAt:   "1906-04-18T13:12:00Z",
			UpdatedAt:   "1906-04-18T13:12:01Z",
		}
		processRepo.ScaleProcessReturns(*updatedProcessRecord, nil)

		newInstances := 10
		var newMemoryMB int64 = 256
		var newDiskMB int64 = 1024

		testScale = &repositories.ProcessScale{
			Instances: &newInstances,
			MemoryMB:  &newMemoryMB,
			DiskMB:    &newDiskMB,
		}

		scaleAppProcessAction = NewScaleAppProcess(appRepo, processRepo)
	})

	JustBeforeEach(func() {
		responseRecord, responseErr = scaleAppProcessAction.Invoke(context.Background(), testClient, testAppGUID, processType, *testScale)
	})

	When("on the happy path", func() {
		It("does not return an error", func() {
			Expect(responseErr).ToNot(HaveOccurred())
		})
		It("fetches the process associated with the GUID", func() {
			Expect(processRepo.FetchProcessCallCount()).ToNot(BeZero())
			_, _, processGUID := processRepo.FetchProcessArgsForCall(0)
			Expect(processGUID).To(Equal(testProcessGUID))
		})

		It("fabricates a ProcessScaleMessage using the inputs and the process GUID and looked-up space", func() {
			Expect(processRepo.ScaleProcessCallCount()).ToNot(BeZero())
			_, _, scaleProcessMessage := processRepo.ScaleProcessArgsForCall(0)
			Expect(scaleProcessMessage.GUID).To(Equal(testProcessGUID))
			Expect(scaleProcessMessage.SpaceGUID).To(Equal(testProcessSpaceGUID))
			Expect(scaleProcessMessage.Instances).To(Equal(testScale.Instances))
			Expect(scaleProcessMessage.DiskMB).To(Equal(testScale.DiskMB))
			Expect(scaleProcessMessage.MemoryMB).To(Equal(testScale.MemoryMB))
		})
		It("transparently returns a record from repositories.ProcessScale", func() {
			Expect(responseRecord).To(Equal(*updatedProcessRecord))
		})
	})

	When("there is an error fetching the process and", func() {
		When("the error is \"not found\"", func() {
			var (
				toReturnErr error
			)
			BeforeEach(func() {
				toReturnErr = repositories.NotFoundError{}
				processRepo.FetchProcessReturns(repositories.ProcessRecord{}, toReturnErr)
			})
			It("returns an empty record", func() {
				Expect(responseRecord).To(Equal(repositories.ProcessRecord{}))
			})
			It("passes through the error", func() {
				Expect(responseErr).To(Equal(toReturnErr))
			})
		})

		When("the error is some other error", func() {
			var (
				toReturnErr error
			)
			BeforeEach(func() {
				toReturnErr = errors.New("some-other-error")
				processRepo.FetchProcessReturns(repositories.ProcessRecord{}, toReturnErr)
			})
			It("returns an empty record", func() {
				Expect(responseRecord).To(Equal(repositories.ProcessRecord{}))
			})
			It("passes through the error", func() {
				Expect(responseErr).To(Equal(toReturnErr))
			})
		})
	})

	When("there is an error updating the process", func() {
		var (
			toReturnErr error
		)
		BeforeEach(func() {
			toReturnErr = errors.New("some-other-error")
			processRepo.ScaleProcessReturns(repositories.ProcessRecord{}, toReturnErr)
		})
		It("returns an empty record", func() {
			Expect(responseRecord).To(Equal(repositories.ProcessRecord{}))
		})
		It("passes through the error", func() {
			Expect(responseErr).To(Equal(toReturnErr))
		})
	})
})
