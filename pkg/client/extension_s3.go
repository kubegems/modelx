package client

import (
	"context"
	"io"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func S3Download(ctx context.Context, location *url.URL, into io.WriterAt) error {
	cred := location.Query().Get("X-Amz-Credential")

	creds := strings.SplitN(cred, "/", 2)

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(creds[0], creds[1], "")),
		config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{URL: location.Scheme + "://" + location.Host}, nil
				},
			),
		))
	if err != nil {
		return err
	}

	cli := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	splites := strings.Split(location.Path, "/")

	bucket, key := "", ""
	for i, val := range splites {
		if val == "" {
			continue
		}
		bucket = splites[i]
		key = strings.Join(splites[i+1:], "/")
		break
	}

	if _, err := manager.NewDownloader(cli).Download(ctx, into, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}); err != nil {
		return err
	}
	return nil
}

func S3Upload(ctx context.Context, location *url.URL, blob *BlobContent) error {
	cfg, err := config.LoadDefaultConfig(ctx, func(lo *config.LoadOptions) error {
		return nil
	})
	if err != nil {
		return err
	}
	_, err = manager.NewUploader(s3.NewFromConfig(cfg)).Upload(ctx, &s3.PutObjectInput{
		Body:          blob.Content,
		Key:           aws.String(""),
		ContentLength: blob.ContentLength,
	})
	return err
}
