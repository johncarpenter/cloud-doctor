package orchestrator

import (
	"context"

	"github.com/elC0mpa/aws-doctor/model"
	"github.com/elC0mpa/aws-doctor/service"
	"github.com/elC0mpa/aws-doctor/utils"
)

func NewService(identityService service.IdentityService, costService service.CostService, resourceService service.ResourceService) *orchestratorService {
	return &orchestratorService{
		identityService: identityService,
		costService:     costService,
		resourceService: resourceService,
	}
}

func (s *orchestratorService) Orchestrate(flags model.Flags) error {
	if flags.Waste {
		return s.wasteWorkflow()
	}

	if flags.Trend {
		return s.trendWorkflow()
	}

	return s.defaultWorkflow()
}

func (s *orchestratorService) defaultWorkflow() error {
	currentMonthData, err := s.costService.GetCurrentMonthCostsByService(context.Background())
	if err != nil {
		return err
	}

	lastMonthData, err := s.costService.GetLastMonthCostsByService(context.Background())
	if err != nil {
		return err
	}

	currentTotalCost, err := s.costService.GetCurrentMonthTotalCosts(context.Background())
	if err != nil {
		return err
	}

	lastTotalCost, err := s.costService.GetLastMonthTotalCosts(context.Background())
	if err != nil {
		return err
	}

	accountInfo, err := s.identityService.GetAccountInfo(context.Background())
	if err != nil {
		return err
	}

	utils.StopSpinner()

	utils.DrawCostTable(accountInfo.AccountID, *lastTotalCost, *currentTotalCost, lastMonthData, currentMonthData, "UnblendedCost")
	return nil
}

func (s *orchestratorService) trendWorkflow() error {
	costInfo, err := s.costService.GetLastSixMonthsCosts(context.Background())
	if err != nil {
		return err
	}

	accountInfo, err := s.identityService.GetAccountInfo(context.Background())
	if err != nil {
		return err
	}

	utils.StopSpinner()

	utils.DrawTrendChart(accountInfo.AccountID, costInfo)

	return nil
}

func (s *orchestratorService) wasteWorkflow() error {
	unusedIPs, err := s.resourceService.GetUnusedIPs(context.Background())
	if err != nil {
		return err
	}

	unusedVolumes, err := s.resourceService.GetUnusedVolumes(context.Background())
	if err != nil {
		return err
	}

	stoppedInstances, attachedVolumes, err := s.resourceService.GetStoppedInstances(context.Background())
	if err != nil {
		return err
	}

	expiringReservations, err := s.resourceService.GetExpiringReservations(context.Background())
	if err != nil {
		return err
	}

	accountInfo, err := s.identityService.GetAccountInfo(context.Background())
	if err != nil {
		return err
	}

	utils.StopSpinner()

	utils.DrawWasteTable(accountInfo.AccountID, unusedIPs, unusedVolumes, attachedVolumes, expiringReservations, stoppedInstances)

	return nil
}
