package actions

import (
	"context"

	"code.cloudfoundry.org/cf-k8s-api/repositories"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

//counterfeiter:generate -o fake -fake-name CFProcessRepository . CFProcessRepository
type CFProcessRepository interface {
	FetchProcess(context.Context, client.Client, string) (repositories.ProcessRecord, error)
	ScaleProcess(context.Context, client.Client, repositories.ScaleProcessMessage) (repositories.ProcessRecord, error)
}

type ScaleProcess struct {
	processRepo  CFProcessRepository
}

func NewScaleProcess(processRepo CFProcessRepository) *ScaleProcess {
	return &ScaleProcess{
		processRepo: processRepo,
	}
}

func (a *ScaleProcess) Invoke(ctx context.Context, client client.Client, processGUID string, scale repositories.ProcessScale) (repositories.ProcessRecord, error) {
	return repositories.ProcessRecord{}, nil
}