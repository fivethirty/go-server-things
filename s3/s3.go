package s3

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/fivethirty/go-server-things/logs"
)

var logger = logs.Default

type S3 struct {
	config Config
	client *s3.Client
}

type Config struct {
	InstanceID string
	Region     string
	S3Bucket   string
}

func New(
	ctx context.Context,
	config Config,
) (*S3, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(config.Region))
	if err != nil {
		return nil, err
	}
	return &S3{
		config: config,
		client: s3.NewFromConfig(cfg),
	}, nil
}

func (s *S3) Upload(ctx context.Context, file *os.File) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.config.S3Bucket),
		Key:    aws.String(fmt.Sprintf("%s/%s", s.config.InstanceID, file.Name())),
		Body:   file,
	})
	if err != nil {
		return err
	}

	path, err := filepath.Abs(file.Name())
	if err != nil {
		return err
	}

	logger.Info(
		"Backed up",
		"file", path,
		"bucket", s.config.S3Bucket,
		"key", s.config.InstanceID,
	)
	return nil
}
