package repositories

import (
	"context"
	"errors"

	workloadsv1alpha1 "code.cloudfoundry.org/cf-k8s-controllers/apis/workloads/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

//+kubebuilder:rbac:groups=workloads.cloudfoundry.org,resources=cfprocesses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=workloads.cloudfoundry.org,resources=cfprocesses/status,verbs=get

type ProcessRecord struct {
	GUID        string
	SpaceGUID   string
	AppGUID     string
	Type        string
	Command     string
	Instances   int
	MemoryMB    int64
	DiskQuotaMB int64
	Ports       []int32
	HealthCheck workloadsv1alpha1.HealthCheck
	Labels      map[string]string
	Annotations map[string]string
	CreatedAt   string
	UpdatedAt   string
}

type ProcessRepository struct{}

func (r *ProcessRepository) FetchProcess(ctx context.Context, client client.Client, processGUID string) (ProcessRecord, error) {

	// TODO: Could look up namespace from guid => namespace cache to do Get
	processList := &workloadsv1alpha1.CFProcessList{}
	err := client.List(ctx, processList)
	if err != nil { // untested
		return ProcessRecord{}, err
	}
	allProcesses := processList.Items
	matches := filterProcessesByMetadataName(allProcesses, processGUID)

	return returnProcess(matches)
}

func (r *ProcessRepository) FetchProcessesForApp(ctx context.Context, client client.Client, appGUID string) ([]ProcessRecord, error) {
	processList := &workloadsv1alpha1.CFProcessList{}
	err := client.List(ctx, processList)
	if err != nil { // untested
		return []ProcessRecord{}, err
	}
	allProcesses := processList.Items
	matches := filterProcessesByAppGUID(allProcesses, appGUID)

	return returnProcesses(matches)
}

func filterProcessesByMetadataName(processes []workloadsv1alpha1.CFProcess, name string) []workloadsv1alpha1.CFProcess {
	var filtered []workloadsv1alpha1.CFProcess
	for i, process := range processes {
		if process.ObjectMeta.Name == name {
			filtered = append(filtered, processes[i])
		}
	}
	return filtered
}

func returnProcess(processes []workloadsv1alpha1.CFProcess) (ProcessRecord, error) {
	if len(processes) == 0 {
		return ProcessRecord{}, NotFoundError{}
	}
	if len(processes) > 1 {
		return ProcessRecord{}, errors.New("duplicate processes exist")
	}

	return cfProcessToProcessRecord(processes[0]), nil
}

func filterProcessesByAppGUID(processes []workloadsv1alpha1.CFProcess, appGUID string) []workloadsv1alpha1.CFProcess {
	var filtered []workloadsv1alpha1.CFProcess
	for i, process := range processes {
		if process.Spec.AppRef.Name == appGUID {
			filtered = append(filtered, processes[i])
		}
	}
	return filtered
}

func returnProcesses(processes []workloadsv1alpha1.CFProcess) ([]ProcessRecord, error) {
	processRecords := make([]ProcessRecord, 0, len(processes))
	for _, process := range processes {
		processRecord := cfProcessToProcessRecord(process)
		processRecords = append(processRecords, processRecord)
	}

	return processRecords, nil
}

func cfProcessToProcessRecord(cfProcess workloadsv1alpha1.CFProcess) ProcessRecord {
	updatedAtTime, _ := getTimeLastUpdatedTimestamp(&cfProcess.ObjectMeta)

	return ProcessRecord{
		GUID:        cfProcess.Name,
		SpaceGUID:   cfProcess.Namespace,
		AppGUID:     cfProcess.Spec.AppRef.Name,
		Type:        cfProcess.Spec.ProcessType,
		Command:     cfProcess.Spec.Command,
		Instances:   cfProcess.Spec.DesiredInstances,
		MemoryMB:    cfProcess.Spec.MemoryMB,
		DiskQuotaMB: cfProcess.Spec.DiskQuotaMB,
		Ports:       cfProcess.Spec.Ports,
		HealthCheck: cfProcess.Spec.HealthCheck,
		Labels:      map[string]string{},
		Annotations: map[string]string{},
		CreatedAt:   cfProcess.CreationTimestamp.UTC().Format(TimestampFormat),
		UpdatedAt:   updatedAtTime,
	}
}
