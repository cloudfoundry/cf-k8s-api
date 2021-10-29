package actions

import (
	"context"

	"code.cloudfoundry.org/cf-k8s-api/repositories"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ScaleAppProcess struct {
	appRepo     CFAppRepository
	processRepo CFProcessRepository
}

func NewScaleAppProcess(appRepo CFAppRepository, processRepo CFProcessRepository) *ScaleAppProcess {
	return &ScaleAppProcess{
		appRepo:     appRepo,
		processRepo: processRepo,
	}
}

func (a *ScaleAppProcess) Invoke(ctx context.Context, client client.Client, appGUID string, processType string, scale repositories.ProcessScale) (repositories.ProcessRecord, error) {
	app, err := a.appRepo.FetchApp(ctx, client, appGUID)
	if err != nil {
		return repositories.ProcessRecord{}, err
	}

	appProcesses, err := a.processRepo.FetchProcessesForApp(ctx, client, app.GUID, app.SpaceGUID)
	if err != nil {
		return repositories.ProcessRecord{}, err
	}

	var appProcessGUID string
	for _, v := range appProcesses {
		if v.Type == processType {
			appProcessGUID = v.GUID
			break
		}
	}

	process, err := a.processRepo.FetchProcess(ctx, client, appProcessGUID)
	
	if err != nil {
		return repositories.ProcessRecord{}, err
	}
	scaleMessage := repositories.ScaleProcessMessage{
		GUID:         process.GUID,
		SpaceGUID:    process.SpaceGUID,
		ProcessScale: scale,
	}
	return a.processRepo.ScaleProcess(ctx, client, scaleMessage)
}
