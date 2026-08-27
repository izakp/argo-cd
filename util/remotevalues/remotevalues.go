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

func IsEmpty(appName string) bool {
	if os.Getenv("REMOTE_VALUES") == "" {
		return true
	}
	return false
}

func Get(appName string) ([]byte, error) {
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
		return nil, fmt.Errorf("Error reading remote values: read S3 object %s/%s: %w", bucket, key, err)
	}
	log.Infof("Sucessfully fetched remote values file: %s", key)

	// Try to get app-specific S3 values file with the same key name on {appName} nested path

	var appSuccess bool
	var appData []byte
	appKey := fmt.Sprintf("%s/%s", appName, key)

	appOut, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(appKey),
	})
	if err != nil {
		log.Warnf("Error fetching remote values (no S3 object): %s/%s: %w", bucket, appKey, err)
		appSuccess = false
	}
	defer appOut.Body.Close()

	if appSuccess {
		appData, err = io.ReadAll(appOut.Body)
		if err != nil {
			log.Warnf("Error reading remote values S3 object: %s/%s: %w", bucket, appKey, err)
			appSuccess = false
		} else {
			log.Infof("Sucessfully fetched remote values file: %s", appKey)
			appSuccess = true
		}
	}

	if appSuccess {
		allData := append(data, appData...)
		return allData, nil
	}
	return data, nil
}
