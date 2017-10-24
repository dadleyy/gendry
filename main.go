package main

import "io"
import "fmt"
import "log"
import "flag"
import "path"
import "bufio"
import "regexp"
import "strings"
import "strconv"
import "net/url"
import "net/http"
import "golang.org/x/tools/cover"

const (
	defaultReportHome    = "http://coverage.marlow.sizethree.cc.s3.amazonaws.com"
	modeIdentifier       = "mode: "
	defaultShieldText    = "generated--coverage"
	defaultShieldStyle   = "flat-square"
	shieldConfigTemplate = "%s-%.2f%%-%s"
	shieldURLTemplate    = "https://img.shields.io/badge/%s.svg"
)

var lineRe = regexp.MustCompile(`^(.+):([0-9]+).([0-9]+),([0-9]+).([0-9]+) ([0-9]+) ([0-9]+)$`)

type server struct {
	reportHome    string
	shieldText    string
	cacheDuration int
	routes        *routeList
}

type action func(http.ResponseWriter, *http.Request, url.Values)

type route interface {
	get(http.ResponseWriter, *http.Request, url.Values)
	post(http.ResponseWriter, *http.Request, url.Values)
}

type routeList map[*regexp.Regexp]route

func (l *routeList) add(routePath *regexp.Regexp, routeHandler route) error {
	if l == nil {
		return fmt.Errorf("invalid-list")
	}

	(*l)[routePath] = routeHandler
	return nil
}

func (l *routeList) actionFor(method string, r route) action {
	switch strings.ToUpper(method) {
	case "POST":
		return r.post
	default:
		return r.get
	}
}

func (l *routeList) Match(request *http.Request) (action, url.Values, bool) {
	path := []byte(request.URL.EscapedPath())

	if l == nil {
		return nil, nil, false
	}

	for re, handler := range *l {
		if match := re.Match(path); match != true {
			continue
		}

		if s := re.NumSubexp(); s == 0 {
			return l.actionFor(request.Method, handler), make(url.Values), true
		}

		groups := re.FindAllStringSubmatch(string(path), -1)
		names := re.SubexpNames()

		if groups == nil || len(groups) != 1 {
			return l.actionFor(request.Method, handler), make(url.Values), true
		}

		values := groups[0][1:]
		params := make(url.Values)
		count := len(names)

		if count >= 0 {
			names = names[1:]
			count = len(names)
		}

		for indx, v := range values {
			if indx < count && len(names[indx]) >= 1 {
				params.Set(names[indx], v)
				continue
			}

			params.Set(fmt.Sprintf("$%d", indx), v)
		}

		return l.actionFor(request.Method, handler), params, true
	}

	return nil, nil, false
}

type reportProfile struct {
	coverage float64
	files    map[string]*cover.Profile
	mode     string
}

type routeWith404 struct {
}

func (route *routeWith404) post(responseWriter http.ResponseWriter, request *http.Request, params url.Values) {
	responseWriter.WriteHeader(404)
}

type badgeAPI struct {
	routeWith404
	reportHome    string
	shieldText    string
	cacheDuration int
}

func (api *badgeAPI) get(responseWriter http.ResponseWriter, request *http.Request, params url.Values) {
	log.Printf("matched badge route, params: %v", params)
	reportName := "latest"

	if commit := request.URL.Query().Get("commit"); commit != "" {
		reportName = commit
	}

	reportURL := fmt.Sprintf("%s/%s", api.reportHome, path.Join(reportName, "library.coverage.txt"))
	r, e := http.Get(reportURL)

	if e != nil {
		log.Printf("error fetching %s: %s", reportURL, e.Error())
		responseWriter.WriteHeader(404)
		return
	}

	defer r.Body.Close()

	report, e := parseProfiles(r.Body)

	if e != nil {
		log.Printf("invalid report: %s", e.Error())
		responseWriter.WriteHeader(404)
		return
	}

	color := "414141"

	if report.coverage > 80 {
		color = "green"
	}

	escapedConfig := url.PathEscape(fmt.Sprintf(shieldConfigTemplate, api.shieldText, report.coverage, color))
	shieldURL, e := url.Parse(fmt.Sprintf(shieldURLTemplate, escapedConfig))

	if e != nil {
		log.Printf("unable to build shield url: %s", e.Error())
		responseWriter.WriteHeader(502)
		return
	}

	shieldQueryParams := url.Values{
		"style": []string{defaultShieldStyle},
	}

	if requestStyle := request.URL.Query().Get("style"); requestStyle != "" {
		shieldQueryParams.Set("style", requestStyle)
	}

	shieldURL.RawQuery = shieldQueryParams.Encode()

	log.Printf("requesting shield: %s", shieldURL)

	client := &http.Client{}
	shieldRequest, e := http.NewRequest("GET", shieldURL.String(), nil)

	if e != nil {
		log.Printf("unable to request shield data: %s", e.Error())
		responseWriter.WriteHeader(502)
		return
	}

	shieldResponse, e := client.Do(shieldRequest)

	if e != nil {
		log.Printf("unable to request shield data: %s", e.Error())
		responseWriter.WriteHeader(502)
		return
	}

	defer shieldResponse.Body.Close()

	cacheValue := fmt.Sprintf("max-age=%d", api.cacheDuration)

	if api.cacheDuration < 0 {
		cacheValue = "no-cache"
	}

	responseWriter.Header().Set("Cache-Control", cacheValue)
	responseWriter.Header().Set("Content-Type", "image/svg+xml")
	responseWriter.WriteHeader(200)
	io.Copy(responseWriter, shieldResponse.Body)
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

	badges := &badgeAPI{
		reportHome:    options.reportHome,
		shieldText:    options.shieldText,
		cacheDuration: options.cacheDuration,
	}

	routes := &routeList{
		regexp.MustCompile("^/reports/(.*)/badge.svg"): badges,
	}

	s := &server{
		routes: routes,
	}

	go func() {
		closed <- http.ListenAndServe(options.address, s)
	}()

	log.Printf("server starting on %s", options.address)
	<-closed
	log.Printf("server terminating")
}

func parseProfiles(r io.Reader) (*reportProfile, error) {
	reader := bufio.NewReader(r)
	scanner := bufio.NewScanner(reader)
	files := make(map[string]*cover.Profile)
	mode := ""

	var total, covered int64

	for scanner.Scan() {
		line := scanner.Text()

		if mode == "" && !strings.HasPrefix(line, modeIdentifier) {
			return nil, fmt.Errorf("invalid-report: [%s]", line)
		}

		if strings.HasPrefix(line, modeIdentifier) {
			mode = line
			continue
		}

		match := lineRe.FindStringSubmatch(line)

		if match == nil {
			return nil, fmt.Errorf("invalid-report: line[%s]", line)
		}

		fileName := match[1]
		existingProfile := files[fileName]

		if existingProfile == nil {
			existingProfile = &cover.Profile{
				FileName: fileName,
				Mode:     mode,
			}

			files[fileName] = existingProfile
		}

		intVals, e := atois(match[2:]...)

		if e != nil {
			return nil, e
		}

		block := cover.ProfileBlock{
			StartLine: intVals[0],
			StartCol:  intVals[1],
			EndLine:   intVals[2],
			EndCol:    intVals[3],
			NumStmt:   intVals[4],
			Count:     intVals[5],
		}

		total += int64(block.NumStmt)

		if block.Count > 0 {
			covered += int64(block.NumStmt)
		}

		existingProfile.Blocks = append(existingProfile.Blocks, block)
	}

	if e := scanner.Err(); e != nil {
		return nil, e
	}

	percent := float64(0)

	if total > 0 {
		percent = float64(covered) / float64(total) * 100
	}

	return &reportProfile{
		coverage: percent,
		files:    files,
		mode:     mode,
	}, nil
}

func atois(strings ...string) ([]int, error) {
	results := []int{}

	for _, v := range strings {
		i, e := strconv.Atoi(v)

		if e != nil {
			return nil, e
		}

		results = append(results, i)
	}

	return results, nil
}
