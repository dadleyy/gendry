package main

import "io"
import "log"
import "flag"
import "regexp"
import "strings"
import "net/http"
import "github.com/dadleyy/gendry/gendry"

const (
	defaultReportHome = "http://coverage.marlow.sizethree.cc.s3.amazonaws.com"
	defaultShieldText = "generated--coverage"
)

type server struct {
	reportHome    string
	shieldText    string
	cacheDuration int
	routes        *gendry.RouteList
}

func (s *server) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	route, params, found := s.routes.Match(request)

	if found {
		route(responseWriter, request, params)
		return
	}

	responseWriter.WriteHeader(404)
	io.Copy(responseWriter, strings.NewReader("not-found"))
}

func (s *server) start(address string, errors chan<- error) {
	errors <- http.ListenAndServe(address, s)
}

func main() {
	options := struct {
		address       string
		reportHome    string
		shieldText    string
		cacheDuration int
	}{}

	flag.StringVar(&options.address, "address", "0.0.0.0:8080", "the address to bind the http listener to")
	flag.StringVar(&options.reportHome, "report-home", defaultReportHome, "where to look for coverage reports")
	flag.StringVar(&options.shieldText, "shield-text", defaultShieldText, "text to display next to percentage")
	flag.IntVar(&options.cacheDuration, "max-cache-age", 10, "amount of seconds for Cache-Control header")
	flag.Parse()

	if options.address == "" {
		log.Fatal("invalid address")
	}

	closed := make(chan error)

	badges := &gendry.BadgeAPI{
		ReportHome:    options.reportHome,
		ShieldText:    options.shieldText,
		CacheDuration: options.cacheDuration,
	}

	routes := &gendry.RouteList{
		regexp.MustCompile("^/reports/(.*)/badge.svg"): badges,
	}

	s := &server{
		routes: routes,
	}

	go s.start(options.address, closed)

	log.Printf("server starting on %s", options.address)
	<-closed
	log.Printf("server terminating")
}
