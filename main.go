package main

import "os"
import "io"
import "fmt"
import "flag"
import "regexp"
import "net/url"
import "log/syslog"
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

type environment func(string) string

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

func (o *cliOptions) env(env environment) error {
	if key := env(constants.AWSAccessKeyIDEnvVariable); key != "" {
		o.awsAccessKeyID = key
	}

	if key := env(constants.AWSAccessKeyEnvVariable); key != "" {
		o.awsAccessKey = key
	}

	if bucket := env(constants.AWSBucketNameEnvVariable); bucket != "" {
		o.awsBucketName = bucket
	}

	if port := env(constants.DatabasePortEnvVariable); port != "" {
		o.databasePort = port
	}

	if host := env(constants.DatabaseHostnameEnvVariable); host != "" {
		o.databaseHostname = host
	}

	if password := env(constants.DatabasePasswordEnvVariable); password != "" {
		o.databasePassword = password
	}

	if user := env(constants.DatabaseUsernameEnvVariable); user != "" {
		o.databaseUsername = user
	}

	if db := env(constants.DatabaseDatabaseEnvVariable); db != "" {
		o.databaseName = db
	}

	return nil
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

type leveledLogger struct {
	output io.Writer
	tag    string
}

func (l *leveledLogger) write(level string, format string, items ...interface{}) {
	message := fmt.Sprintf(format, items...)
	fmt.Fprintf(l.output, "%s [%s] %s\n", level, l.tag, message)
}

func (l *leveledLogger) Debugf(format string, items ...interface{}) {
	l.write("debug", format, items...)
}

func (l *leveledLogger) Infof(format string, items ...interface{}) {
	l.write("info", format, items...)
}

func (l *leveledLogger) Warnf(format string, items ...interface{}) {
	l.write("warn", format, items...)
}

func (l *leveledLogger) Errorf(format string, items ...interface{}) {
	l.write("error", format, items...)
}

func logger(tag string) gendry.LeveledLogger {
	l := leveledLogger{
		output: logOuput,
		tag:    tag,
	}

	return &l
}

var logOuput io.Writer = os.Stdout

func main() {
	godotenv.Load()
	options := cliOptions{}

	if os.Getenv(constants.SyslogAddressEnvVariable) != "" && os.Getenv(constants.SyslogNetworkEnvVariable) != "" {
		addr := os.Getenv(constants.SyslogAddressEnvVariable)
		network := os.Getenv(constants.SyslogNetworkEnvVariable)
		tag := os.Getenv(constants.SyslogTagEnvVariable)
		connection, e := syslog.Dial(network, addr, syslog.LOG_EMERG|syslog.LOG_KERN, tag)

		if e != nil {
			panic(e)
		}

		logOuput = connection

		if e := connection.Info("starting remote gendry syslog..."); e != nil {
			panic(e)
		}

		defer connection.Close()
	}

	log := logger("main")

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
		log.Errorf("invalid address")
		return
	}

	if e := options.env(os.Getenv); e != nil {
		panic(e)
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
		log.Errorf("unable to connect to database: %s", e.Error())
		return
	}

	if e := db.Ping(); e != nil {
		log.Errorf("unable to connect to database: %s", e.Error())
		return
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

	badgeEndpoint := regexp.MustCompile(constants.DisplayAPIRegex)

	routes := &gendry.RouteList{
		badgeEndpoint:                    gendry.NewDisplayAPI(rs, ps, fs),
		regexp.MustCompile("^/reports"):  gendry.NewReportAPI(rs, ps, fs, logger("report api")),
		regexp.MustCompile("^/projects"): gendry.NewProjectAPI(ps, logger("projects api")),
	}

	runtime := gendry.NewRuntime(routes, logger("runtime"))

	go runtime.Start(options.address, closed)

	log.Infof("server starting on %s", options.address)
	<-closed
	log.Infof("server terminating")
}
