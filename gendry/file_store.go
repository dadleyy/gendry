package gendry

import "io"
import "fmt"
import "path"
import "net/url"
import "database/sql"
import "github.com/satori/go.uuid"
import "github.com/aws/aws-sdk-go/aws"
import "github.com/aws/aws-sdk-go/service/s3"
import "github.com/aws/aws-sdk-go/aws/session"
import "github.com/aws/aws-sdk-go/aws/credentials"
import "github.com/aws/aws-sdk-go/service/s3/s3manager"

import "github.com/dadleyy/gendry/gendry/models"
import "github.com/dadleyy/gendry/gendry/constants"

// FileStore defines an interface that is useful for creating writer that persist files.
type FileStore interface {
	NewFile(string, string) (string, io.WriteCloser, error)
	FindFile(string) (io.ReadCloser, error)
}

// NewFileStore returns an implementation of the FileStore interface.
func NewFileStore(driver string, configuration *url.Values, db *sql.DB) FileStore {
	switch driver {
	case "s3":
		store := &s3store{
			accessID:    configuration.Get(constants.AWSAccessKeyIDEnvVariable),
			accessToken: configuration.Get(constants.AWSAccessTokenEnvVariable),
			accessKey:   configuration.Get(constants.AWSAccessKeyEnvVariable),
			bucketName:  configuration.Get(constants.AWSBucketNameEnvVariable),
			persistence: models.NewFileStore(db),
			region:      configuration.Get("AWS_REGION"),
		}
		return store
	default:
		return &tempstore{}
	}
}

type tempstore struct {
}

func (s *tempstore) FindFile(string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("not-implmented")
}

func (s *tempstore) NewFile(string, string) (string, io.WriteCloser, error) {
	return "", nil, fmt.Errorf("not-implmented")
}

type s3store struct {
	accessID    string
	accessToken string
	accessKey   string
	bucketName  string
	region      string
	persistence models.FileStore
}

func (s *s3store) NewFile(contentType string, directory string) (string, io.WriteCloser, error) {
	creds := credentials.NewStaticCredentials(s.accessID, s.accessKey, s.accessToken)

	if _, e := creds.Get(); e != nil {
		return "", nil, fmt.Errorf("invalid-credentials")
	}

	region := s.region

	if region == "" {
		region = "us-east-1"
	}

	config := aws.NewConfig().WithRegion(region).WithCredentials(creds)
	client := s3.New(session.New(), config)

	if _, e := client.ListObjects(&s3.ListObjectsInput{Bucket: &s.bucketName}); e != nil {
		return "", nil, e
	}

	uploadSession, e := session.NewSessionWithOptions(session.Options{
		Config: *config,
	})

	if e != nil {
		return "", nil, e
	}

	uploader := s3manager.NewUploader(uploadSession)

	pr, pw := io.Pipe()
	id := fmt.Sprintf("%s", uuid.NewV4())

	record := models.File{
		SystemID: id,
		Status:   "PENDING",
	}

	if _, e := s.persistence.CreateFiles(record); e != nil {
		return "", nil, e
	}

	go func() {
		key := path.Join(directory, id)

		input := &s3manager.UploadInput{
			Bucket:      aws.String(s.bucketName),
			Key:         aws.String(key),
			ContentType: aws.String(contentType),
			Body:        pr,
		}

		if _, e := uploader.Upload(input); e != nil {
			pr.CloseWithError(fmt.Errorf("unable to put object into s3 (error: %v)", e))
			return
		}

		blueprint := &models.FileBlueprint{
			SystemID: []string{id},
		}

		if _, e, _ := s.persistence.UpdateFileStatus("VALID", blueprint); e != nil {
			pr.CloseWithError(e)
			return
		}

		pr.Close()
	}()

	return id, pw, nil
}

func (s *s3store) FindFile(systemID string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("not-implemented")
}
