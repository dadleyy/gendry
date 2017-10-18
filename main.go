package main

import "io"
import "fmt"
import "log"
import "flag"
import "path"
import "bufio"
import "bytes"
import "regexp"
import "strings"
import "net/http"

// import "github.com/golang/tools/cover"

const (
	defaultReportHome = "http://coverage.marlow.sizethree.cc.s3.amazonaws.com"
	modeIdentifier    = "mode: "
)

var lineRe = regexp.MustCompile(`^(.+):([0-9]+).([0-9]+),([0-9]+).([0-9]+) ([0-9]+) ([0-9]+)$`)

type server struct {
	reportHome string
}

func (s *server) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	reportName := "latest"

	if commit := request.URL.Query().Get("commit"); commit != "" {
		reportName = commit
	}

	reportUrl := fmt.Sprintf("%s/%s", s.reportHome, path.Join(reportName, "library.coverage.txt"))
	r, e := http.Get(reportUrl)

	if e != nil {
		log.Printf("error fetching %s: %s", reportUrl, e.Error())
		responseWriter.WriteHeader(404)
		return
	}

	defer r.Body.Close()

	reader := bufio.NewReader(r.Body)
	scanner := bufio.NewScanner(reader)
	responseBuffer := new(bytes.Buffer)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, modeIdentifier) {
			continue
		}

		match := lineRe.FindStringSubmatch(line)

		if match == nil {
			log.Printf("invalid file detected: %s", reportUrl)
			responseWriter.WriteHeader(422)
			return
		}

		fmt.Fprintf(responseBuffer, "%s\n", line)
	}

	responseWriter.WriteHeader(200)
	io.Copy(responseWriter, responseBuffer)
}

func main() {
	options := struct {
		address    string
		reportHome string
	}{}

	flag.StringVar(&options.address, "address", "0.0.0.0:8080", "the address to bind the http listener to")
	flag.StringVar(&options.reportHome, "report-home", defaultReportHome, "where to look for coverage reports")
	flag.Parse()

	if options.address == "" {
		log.Fatal("invalid address")
	}

	closed := make(chan error)

	go func() {
		s := &server{
			reportHome: options.reportHome,
		}

		closed <- http.ListenAndServe(options.address, s)
	}()

	log.Printf("server starting on %s", options.address)
	<-closed
	log.Printf("server terminating")
}

/*
	files := make(map[string]*Profile)
	buf := bufio.NewReader(pf)
	// First line is "mode: foo", where foo is "set", "count", or "atomic".
	// Rest of file is in the format
	//	encoding/base64/base64.go:34.44,37.40 3 1
	// where the fields are: name.go:line.column,line.column numberOfStatements count
	s := bufio.NewScanner(buf)
	mode := ""
	for s.Scan() {
		line := s.Text()
		if mode == "" {
			const p = "mode: "
			if !strings.HasPrefix(line, p) || line == p {
				return nil, fmt.Errorf("bad mode line: %v", line)
			}
			mode = line[len(p):]
			continue
		}
		m := lineRe.FindStringSubmatch(line)
		if m == nil {
			return nil, fmt.Errorf("line %q doesn't match expected format: %v", line, lineRe)
		}
		fn := m[1]
		p := files[fn]
		if p == nil {
			p = &Profile{
				FileName: fn,
				Mode:     mode,
			}
			files[fn] = p
		}
		p.Blocks = append(p.Blocks, ProfileBlock{
			StartLine: toInt(m[2]),
			StartCol:  toInt(m[3]),
			EndLine:   toInt(m[4]),
			EndCol:    toInt(m[5]),
			NumStmt:   toInt(m[6]),
			Count:     toInt(m[7]),
		})
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	for _, p := range files {
		sort.Sort(blocksByStart(p.Blocks))
	}
	// Generate a sorted slice.
	profiles := make([]*Profile, 0, len(files))
	for _, profile := range files {
		profiles = append(profiles, profile)
	}
	sort.Sort(byFileName(profiles))
	return profiles, nil
*/
