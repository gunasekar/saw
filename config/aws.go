package config

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

type AWSConfiguration struct {
	Endpoint string
	Region   string
	Profile  string
}

func (c *AWSConfiguration) LoadConfig() (aws.Config, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(c.Region),
	}

	if c.Profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(c.Profile))
	}

	return config.LoadDefaultConfig(context.TODO(), opts...)
}
