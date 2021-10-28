package payloads

import "code.cloudfoundry.org/cf-k8s-api/repositories"

type ProcessScale struct {
	Instances *int `json:"instances"`
	MemoryMB  *int `json:"memory_in_mb"`
	DiskMB    *int `json:"disk_in_mb"`
}

func (p ProcessScale) ToRecord() repositories.ProcessScale {
	return repositories.ProcessScale{
		Instances: p.Instances,
		MemoryMB:  p.MemoryMB,
		DiskMB:    p.DiskMB,
	}
}