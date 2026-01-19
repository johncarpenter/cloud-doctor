package orchestrator

import (
	"github.com/elC0mpa/aws-doctor/model"
	"github.com/elC0mpa/aws-doctor/service"
)

type orchestratorService struct {
	identityService service.IdentityService
	costService     service.CostService
	resourceService service.ResourceService
}

type OrchestratorService interface {
	Orchestrate(model.Flags) error
}
