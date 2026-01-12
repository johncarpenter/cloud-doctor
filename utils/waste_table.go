package utils

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/elC0mpa/aws-billing/model"
)

func DrawWasteTable(accountId string, elasticIpInfo []types.Address, unusedEBSVolumeInfo []types.Volume, attachedToStoppedInstancesEBSVolumeInfo []string, expireReservedInstancesInfo []model.RiExpirationInfo, instancesStoppedMoreThan30Days []types.Instance) {
}
