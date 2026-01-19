package awssts

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/elC0mpa/aws-doctor/model"
)

type service struct {
	client *sts.Client
}

type STSService interface {
	GetCallerIdentity(ctx context.Context) (*sts.GetCallerIdentityOutput, error)
	GetAccountInfo(ctx context.Context) (*model.AccountInfo, error)
}
