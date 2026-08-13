package backup

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// AWSPutClient adapts the AWS SDK S3 client to ObjectPutter.
type AWSPutClient struct {
	Client *s3.Client
}

func (c *AWSPutClient) PutObject(ctx context.Context, bucket, key string, body io.Reader, size int64, contentType string) error {
	if c == nil || c.Client == nil {
		return fmt.Errorf("aws s3 client nil")
	}
	input := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   body,
	}
	if size >= 0 {
		input.ContentLength = aws.Int64(size)
	}
	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}
	_, err := c.Client.PutObject(ctx, input)
	return err
}

// NewAWSSinkFromEnv loads default AWS config only when bucket is non-empty.
// Callers must not invoke this when the bucket is unset (ignore ambient creds).
func NewAWSSinkFromEnv(ctx context.Context, bucket, region, endpoint string) (*S3Sink, error) {
	if bucket == "" {
		return nil, fmt.Errorf("bucket required")
	}
	var opts []func(*awsconfig.LoadOptions) error
	if region != "" {
		opts = append(opts, awsconfig.WithRegion(region))
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, err
	}
	var s3opts []func(*s3.Options)
	if endpoint != "" {
		s3opts = append(s3opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		})
	}
	client := s3.NewFromConfig(cfg, s3opts...)
	return NewS3Sink(&AWSPutClient{Client: client}, bucket), nil
}
