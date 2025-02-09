package s3

import (
	"context"
	"os"
	"path/filepath"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/fivethirty/go-server-things/logs"
)

type Backup struct {
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
) (*Backup, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(config.Region))
	if err != nil {
		return nil, err
	}
	return &Backup{
		config: config,
		client: s3.NewFromConfig(cfg),
	}, nil
}

var logger = logs.Default

func (b *Backup) Upload(ctx context.Context, file *os.File) error {
	_, err := b.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(b.config.S3Bucket),
		Key:    aws.String(b.config.InstanceID),
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
		"bucket", b.config.S3Bucket,
		"key", b.config.InstanceID,
	)
	return nil
}
