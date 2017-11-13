package main

import "io"
import "log"
import "fmt"
import "flag"
import "regexp"
import "strings"
import "net/http"
import "database/sql"
import "github.com/go-sql-driver/mysql"
import "github.com/dadleyy/gendry/gendry"
import "github.com/dadleyy/gendry/gendry/models"

const (
	defaultReportHome  = "http://coverage.marlow.sizethree.cc.s3.amazonaws.com"
	dbConnectionString = "user=%s password=%s host=%s port=%s dbname=%s sslmode=%s"
	defaultDatbaseUser = "gendry"
	defaultDatbasePass = "password"
	defaultDatbaseName = "gendry"
	defaultDatbaseHost = "0.0.0.0"
	defaultDatbasePort = "3306"
)

type server struct {
	cacheDuration int
	routes        *gendry.RouteList
}

func (s *server) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	route, params, found := s.routes.Match(request)

	if found {
		route(responseWriter, request, params)
		return
	}

	log.Printf("not found: %v", request.URL)
	responseWriter.WriteHeader(404)
	io.Copy(responseWriter, strings.NewReader("not-found"))
}

func (s *server) start(address string, errors chan<- error) {
	errors <- http.ListenAndServe(address, s)
}

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
	flag.Parse()

	if options.address == "" {
		log.Fatal("invalid address")
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

	ps := models.NewProjectStore(db)
	rs := models.NewReportStore(db)

	defer db.Close()

	routes := &gendry.RouteList{
		regexp.MustCompile("^/reports/(?P<project>.*)/(?P<tag>.*)/badge.svg"): gendry.NewBadgeAPI(),
		regexp.MustCompile("^/reports"):                                       gendry.NewReportAPI(rs, ps),
		regexp.MustCompile("^/projects"):                                      gendry.NewProjectAPI(ps),
	}

	s := &server{
		routes: routes,
	}

	go s.start(options.address, closed)

	log.Printf("server starting on %s", options.address)
	<-closed
	log.Printf("server terminating")
}
