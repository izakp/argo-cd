package v1alpha1

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	log "github.com/sirupsen/logrus"
	reflect "reflect"
	"strings"

	runtime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Set the ValuesObject property to the json representation of the yaml contained in value
// Remove Values property if present
func (h *ApplicationSourceHelm) SetValuesString(value string) error {
	if value == "" {
		h.ValuesObject = nil
		h.Values = ""
	} else {
		data, err := yaml.YAMLToJSON([]byte(value))
		if err != nil {
			return fmt.Errorf("failed converting yaml to json: %v", err)
		}
		var v interface{}
		if err := json.Unmarshal(data, &v); err != nil {
			return fmt.Errorf("failed to unmarshal json: %v", err)
		}
		switch v.(type) {
		case string:
		case map[string]interface{}:
		default:
			return fmt.Errorf("invalid type %q", reflect.TypeOf(v))
		}
		h.ValuesObject = &runtime.RawExtension{Raw: data}
		h.Values = ""
	}
	return nil
}

func (h *ApplicationSourceHelm) RemoteValuesIsEmpty() bool {
	if os.Getenv("REMOTE_VALUES") == "" {
		return true
	}
	return false
}

func (h *ApplicationSourceHelm) GetRemoteValues() ([]byte, error) {
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

func (h *ApplicationSourceHelm) ValuesYAML() []byte {
	if h.ValuesObject == nil || h.ValuesObject.Raw == nil {
		return []byte(h.Values)
	}
	b, err := yaml.JSONToYAML(h.ValuesObject.Raw)
	if err != nil {
		// This should be impossible, because rawValue isn't set directly.
		return []byte{}
	}
	return b
}

func (h *ApplicationSourceHelm) ValuesIsEmpty() bool {
	return len(h.ValuesYAML()) == 0
}

func (h *ApplicationSourceHelm) ValuesString() string {
	if h.ValuesObject == nil || h.ValuesObject.Raw == nil {
		return h.Values
	}
	return strings.TrimSuffix(string(h.ValuesYAML()), "\n")
}
