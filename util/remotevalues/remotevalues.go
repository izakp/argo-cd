package remotevalues

import (
	"io"
	"os"
	"net/url"
	"strings"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func IsEmpty() bool {
	if os.Getenv("REMOTE_VALUES") == "" {
		return true
	}
	return false
}

func Get() ([]byte, error) {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}

	s3URI := os.Getenv("REMOTE_VALUES")
	parsed, err := url.Parse(s3URI)
	if err != nil {
		return nil, fmt.Errorf("Error fetching remote values: parse S3 URI %q: %w", s3URI, err)
	}
	if parsed.Scheme != "s3" {
		return nil, fmt.Errorf("Error fetching remote values: invalid S3 URI scheme %q, expected s3", parsed.Scheme)
	}
	log.Infof("Parsed remote values file: %s", s3URI)

	bucket := parsed.Host
	key := strings.TrimPrefix(parsed.Path, "/")
	if bucket == "" || key == "" {
		return nil, fmt.Errorf("Error fetching remote values: invalid S3 URI %q, expected s3://bucket/key", s3URI)
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		return nil, fmt.Errorf("Error fetching remote values: create AWS session: %w", err)
	}

	svc := s3.New(sess)

	out, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("Error fetching remote values: get S3 object %s/%s: %w", bucket, key, err)
	}
	defer out.Body.Close()

	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("Error fetching remote values: read S3 object %s/%s: %w", bucket, key, err)
	}
	log.Infof("Sucessfully fetched remote values file: %s", s3URI)

	return data, nil
}
