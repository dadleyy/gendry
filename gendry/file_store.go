package gendry

import "io"
import "log"
import "fmt"
import "path"
import "bytes"
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
	uploadSession, e := s.newSession()

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

func (s *s3store) FindFile(filepath string) (io.ReadCloser, error) {
	downloadSession, e := s.newSession()

	if e != nil {
		return nil, e
	}

	downloader := s3manager.NewDownloader(downloadSession)
	pr, pw := io.Pipe()

	go func() {
		buffer := make([]byte, 0, constants.MaxHTMLReportFileSize)
		writer := aws.NewWriteAtBuffer(buffer)
		_, e := downloader.Download(writer, &s3.GetObjectInput{
			Bucket: aws.String(s.bucketName),
			Key:    aws.String(filepath),
		})

		if e != nil {
			log.Printf("unable to download from s3: %s", e)
			pw.CloseWithError(e)
			return
		}

		_, e = io.Copy(pw, bytes.NewBuffer(writer.Bytes()))
		pw.CloseWithError(e)
	}()

	return pr, nil
}

func (s *s3store) newSession() (*session.Session, error) {
	creds := credentials.NewStaticCredentials(s.accessID, s.accessKey, s.accessToken)

	if _, e := creds.Get(); e != nil {
		return nil, e
	}

	region := s.region

	if region == "" {
		region = "us-east-1"
	}

	config := aws.NewConfig().WithRegion(region).WithCredentials(creds)

	return session.NewSessionWithOptions(session.Options{Config: *config})
}
