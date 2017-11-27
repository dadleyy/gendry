package gendry

import "io"
import "fmt"
import "bufio"
import "regexp"
import "strings"
import "strconv"
import "golang.org/x/tools/cover"

const (
	modeIdentifier = "mode: "
)

var lineRe = regexp.MustCompile(`^(.+):([0-9]+).([0-9]+),([0-9]+).([0-9]+) ([0-9]+) ([0-9]+)$`)

type reportProfile struct {
	coverage float64
	files    map[string]*cover.Profile
	mode     string
}

func parseCoverProfile(r io.Reader) (*reportProfile, error) {
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
