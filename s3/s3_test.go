package s3_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/fivethirty/go-server-things/s3"
	"github.com/google/uuid"
)

type testClient struct {
	lastPutObjectInput *awss3.PutObjectInput
}

func (tc *testClient) PutObject(
	_ context.Context,
	input *awss3.PutObjectInput,
	_ ...func(*awss3.Options),
) (*awss3.PutObjectOutput, error) {
	tc.lastPutObjectInput = input
	return nil, nil
}

func TestUpload(t *testing.T) {
	t.Parallel()
	uuid := uuid.New()
	dir := fmt.Sprintf("/tmp/%s/", uuid)
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(dir, "test-file"))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		instance string
		bucket   string
		file     func(t *testing.T) *os.File
	}{
		{
			name:     "should upload file",
			instance: "test-instance",
			bucket:   "test-bucket",
			file: func(t *testing.T) *os.File {
				return file
			},
		},
		{
			name:     "should upload file with weird path",
			instance: "other-instance",
			bucket:   "other-bucket",
			file: func(t *testing.T) *os.File {
				t.Helper()
				file, err := os.Open(fmt.Sprintf("/tmp/%s/../%s/test-file", uuid, uuid))
				if err != nil {
					t.Fatal(err)
				}
				return file
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tc := testClient{}

			s3 := s3.NewWithClient(
				s3.Config{
					InstanceID: tt.instance,
					Region:     "us-west-2",
					S3Bucket:   tt.bucket,
				},
				&tc,
				slog.Default(),
			)

			file := tt.file(t)

			err := s3.Upload(context.Background(), file)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			expected := awss3.PutObjectInput{
				Bucket: aws.String(tt.bucket),
				Key:    aws.String(fmt.Sprintf("%s/test-file", tt.instance)),
				Body:   file,
			}

			if *expected.Bucket != *tc.lastPutObjectInput.Bucket {
				t.Fatalf("expected %s but got %s", *expected.Bucket, *tc.lastPutObjectInput.Bucket)
			}

			if *expected.Key != *tc.lastPutObjectInput.Key {
				t.Fatalf("expected %s but got %s", *expected.Key, *tc.lastPutObjectInput.Key)
			}

			if expected.Body != tc.lastPutObjectInput.Body {
				t.Fatalf("expected %v but got %v", expected.Body, tc.lastPutObjectInput.Body)
			}
		})
	}
}
