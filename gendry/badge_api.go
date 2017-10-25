package gendry

import "io"
import "log"
import "fmt"
import "path"
import "bufio"
import "regexp"
import "strings"
import "strconv"
import "net/url"
import "net/http"
import "golang.org/x/tools/cover"

const (
	defaultShieldStyle   = "flat-square"
	modeIdentifier       = "mode: "
	shieldConfigTemplate = "%s-%.2f%%-%s"
	shieldURLTemplate    = "https://img.shields.io/badge/%s.svg"
)

var lineRe = regexp.MustCompile(`^(.+):([0-9]+).([0-9]+),([0-9]+).([0-9]+) ([0-9]+) ([0-9]+)$`)

type reportProfile struct {
	coverage float64
	files    map[string]*cover.Profile
	mode     string
}

// BadgeAPI is resposnible for writing the svg badge result from shields.io given a report name.
type BadgeAPI struct {
	notImplementedRoute
	ReportHome    string
	ShieldText    string
	CacheDuration int
}

// Get response to http GET methods for the BadgeAPI endpoint.
func (api *BadgeAPI) Get(responseWriter http.ResponseWriter, request *http.Request, params url.Values) {
	log.Printf("matched badge route, params: %v", params)
	reportName := "latest"

	if commit := request.URL.Query().Get("commit"); commit != "" {
		reportName = commit
	}

	reportURL := fmt.Sprintf("%s/%s", api.ReportHome, path.Join(reportName, "library.coverage.txt"))
	r, e := http.Get(reportURL)

	if e != nil {
		log.Printf("error fetching %s: %s", reportURL, e.Error())
		responseWriter.WriteHeader(404)
		return
	}

	defer r.Body.Close()

	report, e := api.parseProfiles(r.Body)

	if e != nil {
		log.Printf("invalid report: %s", e.Error())
		responseWriter.WriteHeader(404)
		return
	}

	color := "414141"

	if report.coverage > 80 {
		color = "green"
	}

	escapedConfig := url.PathEscape(fmt.Sprintf(shieldConfigTemplate, api.ShieldText, report.coverage, color))
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

	cacheValue := fmt.Sprintf("max-age=%d", api.CacheDuration)

	if api.CacheDuration < 0 {
		cacheValue = "no-cache"
	}

	responseWriter.Header().Set("Cache-Control", cacheValue)
	responseWriter.Header().Set("Content-Type", "image/svg+xml")
	responseWriter.WriteHeader(200)
	io.Copy(responseWriter, shieldResponse.Body)
}

func (api *BadgeAPI) parseProfiles(r io.Reader) (*reportProfile, error) {
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
