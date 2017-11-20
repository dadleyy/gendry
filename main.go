package main

import "os"
import "log"
import "fmt"
import "flag"
import "regexp"
import "net/url"
import "database/sql"
import "github.com/joho/godotenv"
import "github.com/go-sql-driver/mysql"
import "github.com/dadleyy/gendry/gendry"
import "github.com/dadleyy/gendry/gendry/models"
import "github.com/dadleyy/gendry/gendry/constants"

const (
	defaultReportHome  = "http://coverage.marlow.sizethree.cc.s3.amazonaws.com"
	dbConnectionString = "user=%s password=%s host=%s port=%s dbname=%s sslmode=%s"
	defaultDatbaseUser = "gendry"
	defaultDatbasePass = "password"
	defaultDatbaseName = "gendry"
	defaultDatbaseHost = "0.0.0.0"
	defaultDatbasePort = "3306"
)

type cliOptions struct {
	address          string
	reportHome       string
	cacheDuration    int
	databaseUsername string
	databasePassword string
	databaseHostname string
	databasePort     string
	databaseName     string
	databaseSSL      string
	awsAccessKeyID   string
	awsAccessKey     string
	awsAccessToken   string
	awsBucketName    string
}

func (o *cliOptions) ConnectionString() string {
	params := []interface{}{
		o.databaseUsername,
		o.databasePassword,
		o.databaseHostname,
		o.databasePort,
		o.databaseName,
		o.databaseSSL,
	}

	return fmt.Sprintf(dbConnectionString, params...)
}

func main() {
	godotenv.Load()
	options := cliOptions{}

	flag.StringVar(&options.address, "address", "0.0.0.0:8080", "the address to bind the http listener to")
	flag.StringVar(&options.reportHome, "report-home", defaultReportHome, "where to look for coverage reports")
	flag.IntVar(&options.cacheDuration, "max-cache-age", 10, "amount of seconds for Cache-Control header")
	flag.StringVar(&options.databaseUsername, "db-user", defaultDatbaseUser, "database username")
	flag.StringVar(&options.databaseHostname, "db-host", defaultDatbaseHost, "hostname for database connection")
	flag.StringVar(&options.databasePassword, "db-pass", defaultDatbasePass, "password for database connection")
	flag.StringVar(&options.databasePort, "db-port", defaultDatbasePort, "port where mysql is running")
	flag.StringVar(&options.databaseName, "db-name", defaultDatbaseName, "the database name to connect to")
	flag.StringVar(&options.databaseSSL, "db-ssl", "disable", "enable database ssl")
	flag.StringVar(&options.awsAccessKeyID, "aws-access-key-id", "", "aws access key id")
	flag.StringVar(&options.awsAccessKey, "aws-access-key", "", "aws access key")
	flag.StringVar(&options.awsAccessToken, "aws-access-token", "", "aws access token")
	flag.StringVar(&options.awsBucketName, "aws-bucket-name", "", "aws access token")
	flag.Parse()

	if options.address == "" {
		log.Fatal("invalid address")
	}

	if key := os.Getenv(constants.AWSAccessKeyIDEnvVariable); key != "" {
		options.awsAccessKeyID = key
	}

	if key := os.Getenv(constants.AWSAccessKeyEnvVariable); key != "" {
		options.awsAccessKey = key
	}

	config := mysql.Config{
		User:   options.databaseUsername,
		Passwd: options.databasePassword,
		DBName: options.databaseName,
		Net:    "tcp",
		Addr:   fmt.Sprintf("%s:%s", options.databaseHostname, options.databasePort),
	}

	db, e := sql.Open("mysql", config.FormatDSN())

	if e != nil {
		log.Fatalf("unable to connect to database: %s", e.Error())
	}

	if e := db.Ping(); e != nil {
		log.Fatalf("unable to connect to database: %s", e.Error())
	}

	closed := make(chan error)
	fileStoreConfig := &url.Values{
		constants.AWSAccessKeyEnvVariable:   []string{options.awsAccessKey},
		constants.AWSAccessTokenEnvVariable: []string{options.awsAccessToken},
		constants.AWSAccessKeyIDEnvVariable: []string{options.awsAccessKeyID},
		constants.AWSBucketNameEnvVariable:  []string{options.awsBucketName},
	}

	ps := models.NewProjectStore(db)
	rs := models.NewReportStore(db)

	fs := gendry.NewFileStore("s3", fileStoreConfig, db)

	defer db.Close()

	badgeEndpoint := regexp.MustCompile("^/reports/(?P<project>.*)/(?P<tag>.*)/badge.svg")

	routes := &gendry.RouteList{
		badgeEndpoint:                    gendry.NewBadgeAPI(),
		regexp.MustCompile("^/reports"):  gendry.NewReportAPI(rs, ps, fs),
		regexp.MustCompile("^/projects"): gendry.NewProjectAPI(ps),
	}

	runtime := gendry.NewRuntime(routes)

	go runtime.Start(options.address, closed)

	log.Printf("server starting on %s", options.address)
	<-closed
	log.Printf("server terminating")
}
