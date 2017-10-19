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
	shieldConfigTemplate = "%s-%.2f%%-%s"
	shieldURLTemplate    = "https://img.shields.io/badge/%s.svg"
)

var lineRe = regexp.MustCompile(`^(.+):([0-9]+).([0-9]+),([0-9]+).([0-9]+) ([0-9]+) ([0-9]+)$`)

type server struct {
	reportHome string
	shieldText string
}

type reportProfile struct {
	coverage float64
	files    map[string]*cover.Profile
	mode     string
}

func (s *server) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	reportName := "latest"

	if commit := request.URL.Query().Get("commit"); commit != "" {
		reportName = commit
	}

	reportURL := fmt.Sprintf("%s/%s", s.reportHome, path.Join(reportName, "library.coverage.txt"))
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

	escapedConfig := url.PathEscape(fmt.Sprintf(shieldConfigTemplate, s.shieldText, report.coverage, color))
	shieldURL := fmt.Sprintf(shieldURLTemplate, escapedConfig)
	shieldResponse, e := http.Get(shieldURL)

	if e != nil {
		responseWriter.WriteHeader(502)
		return
	}

	defer shieldResponse.Body.Close()

	responseWriter.Header().Set("Content-Type", "image/svg+xml")
	responseWriter.WriteHeader(200)
	io.Copy(responseWriter, shieldResponse.Body)
}

func main() {
	options := struct {
		address    string
		reportHome string
		shieldText string
	}{}

	flag.StringVar(&options.address, "address", "0.0.0.0:8080", "the address to bind the http listener to")
	flag.StringVar(&options.reportHome, "report-home", defaultReportHome, "where to look for coverage reports")
	flag.StringVar(&options.shieldText, "shield-text", defaultShieldText, "text to display next to percentage")
	flag.Parse()

	if options.address == "" {
		log.Fatal("invalid address")
	}

	closed := make(chan error)

	go func() {
		s := &server{
			reportHome: options.reportHome,
			shieldText: options.shieldText,
		}

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
			return nil, fmt.Errorf("invalid-report")
		}

		if strings.HasPrefix(line, modeIdentifier) {
			mode = line
			continue
		}

		match := lineRe.FindStringSubmatch(line)

		if match == nil {
			return nil, fmt.Errorf("invalid-report")
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
