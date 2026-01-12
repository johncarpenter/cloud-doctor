package main

import (
	"context"

	awsconfig "github.com/elC0mpa/aws-billing/service/aws_config"
	awscostexplorer "github.com/elC0mpa/aws-billing/service/costexplorer"
	awsec2 "github.com/elC0mpa/aws-billing/service/ec2"
	"github.com/elC0mpa/aws-billing/service/flag"
	"github.com/elC0mpa/aws-billing/service/orchestrator"
	awssts "github.com/elC0mpa/aws-billing/service/sts"
	"github.com/elC0mpa/aws-billing/utils"
)

func main() {
	utils.DrawBanner()
	utils.StartSpinner()

	flagService := flag.NewService()
	flags, err := flagService.GetParsedFlags()
	if err != nil {
		panic(err)
	}

	cfgService := awsconfig.NewService()
	awsCfg, err := cfgService.GetAWSCfg(context.Background(), flags.Region, flags.Profile)
	if err != nil {
		panic(err)
	}

	costService := awscostexplorer.NewService(awsCfg)
	stsService := awssts.NewService(awsCfg)
	ec2Service := awsec2.NewService(awsCfg)

	orchestratorService := orchestrator.NewService(stsService, costService, ec2Service)

	err = orchestratorService.Orchestrate(flags)
	if err != nil {
		panic(err)
	}
}
