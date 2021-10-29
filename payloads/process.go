package payloads

import "code.cloudfoundry.org/cf-k8s-api/repositories"

type ProcessScale struct {
	Instances *int   `json:"instances" validate:"omitempty,gte=0"`
	MemoryMB  *int64 `json:"memory_in_mb" validate:"omitempty,gt=0"`
	DiskMB    *int64 `json:"disk_in_mb" validate:"omitempty,gt=0"`
}

func (p ProcessScale) ToRecord() repositories.ProcessScale {
	return repositories.ProcessScale{
		Instances: p.Instances,
		MemoryMB:  p.MemoryMB,
		DiskMB:    p.DiskMB,
	}
}
