package awssts

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/elC0mpa/aws-doctor/model"
)

func NewService(awsconfig aws.Config) *service {
	client := sts.NewFromConfig(awsconfig)
	return &service{
		client: client,
	}
}

func (s *service) GetCallerIdentity(ctx context.Context) (*sts.GetCallerIdentityOutput, error) {
	input := &sts.GetCallerIdentityInput{}

	return s.client.GetCallerIdentity(ctx, input)
}

// GetAccountInfo implements service.IdentityService
func (s *service) GetAccountInfo(ctx context.Context) (*model.AccountInfo, error) {
	output, err := s.GetCallerIdentity(ctx)
	if err != nil {
		return nil, err
	}

	return &model.AccountInfo{
		Provider:    "aws",
		AccountID:   aws.ToString(output.Account),
		AccountName: aws.ToString(output.Arn),
	}, nil
}
