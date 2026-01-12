package orchestrator

import (
	"github.com/elC0mpa/aws-billing/model"
	awscostexplorer "github.com/elC0mpa/aws-billing/service/costexplorer"
	awsec2 "github.com/elC0mpa/aws-billing/service/ec2"
	awssts "github.com/elC0mpa/aws-billing/service/sts"
)

type service struct {
	stsService  awssts.STSService
	costService awscostexplorer.CostService
	ec2Service  awsec2.EC2Service
}

type OrchestratorService interface {
	Orchestrate(model.Flags) error
}
