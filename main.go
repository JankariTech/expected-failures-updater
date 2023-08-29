package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/dpakach/gorkin/lexer"
	"github.com/dpakach/gorkin/object"
	"github.com/dpakach/gorkin/parser"
	"github.com/dpakach/gorkin/token"
)

type scenarioType string

const version = "1.0.0"

const (
	// Outline is scenario Outline type
	Outline scenarioType = "Outline"
	// Normal is regular scenario type
	Normal = "Normal"
)

// Feature is gherkin Feature
type Feature struct {
	Type       scenarioType       `json:"type"`
	LineNumber int                `json:"line_number"`
	Title      string             `json:"title"`
	DataRow    []object.TableData `json:"data_row"`
	FilePath   string             `json:"file_path"`
	Scenario   *object.Scenario   `json:"-"`
	Background *object.Background `json:"-"`
}

type shift struct {
	oldPath string
	newPath string
}

func getShifts(out io.Writer) int {
	latestFeatures := getFeatures()

	oldFeatureData, err := ioutil.ReadFile("output.json")
	if err != nil {
		panic(err)
	}

	var oldFeatures []Feature
	var shifts []shift

	_ = json.Unmarshal(oldFeatureData, &oldFeatures)

	for _, l := range latestFeatures {
		found := false
		for _, o := range oldFeatures {
			if o.Type == Outline {
				if o.Title == l.Title && dataRowSame(l.DataRow, o.DataRow) && getTestSuite(o) == getTestSuite(l) {
					found = true
				}
			} else {
				if o.Title == l.Title && getTestSuite(o) == getTestSuite(l) {
					found = true
				}
			}
			if found {
				if o.LineNumber != l.LineNumber {
					_, _ = io.WriteString(out, "Found Shift\n")
					_, _ = io.WriteString(out, "New: "+getTestPath(l)+"\n")
					_, _ = io.WriteString(out, "\n\n")

					shifts = append(shifts, shift{getTestPath(o), getTestPath(l)})
				}
				break
			}
		}

		if !found {
			_, _ = io.WriteString(out, "new scenario found\n")
			_, _ = io.WriteString(out, "New: "+getTestPath(l))
			_, _ = io.WriteString(out, "\n\n")
		}
	}

	expectedFailuresDir := os.Getenv("EXPECTED_FAILURES_DIR")
	if expectedFailuresDir == "" {
		_, _ = io.WriteString(out, "Expected Failures directory not provided\n")
		_, _ = io.WriteString(out, "Please use EXPECTED_FAILURES_DIR env variable to provide path to directory where expected failure files are\n")
		_, _ = io.WriteString(out, "\n")
		return 1
	}

	expectedFailuresPrefix := os.Getenv("EXPECTED_FAILURES_PREFIX")

	if expectedFailuresPrefix == "" {
		_, _ = io.WriteString(out, "Expected Failures prefix not provided\n")
		_, _ = io.WriteString(out, "using 'expected-failures' as prefix of expected failure files by default\n")
		expectedFailuresPrefix = "expected-failures"
	}

	files, err := ioutil.ReadDir(expectedFailuresDir)
	if err != nil {
		_, _ = io.WriteString(out, err.Error())
		return 1
	}

	for _, f := range files {
		if strings.HasPrefix(f.Name(), expectedFailuresPrefix) {
			err = replaceOccurrenceInFile(shifts, filepath.Join(expectedFailuresDir, f.Name()))
			if err != nil {
				_, _ = io.WriteString(out, err.Error())
				return 1
			}
		}
	}
	return 0
}

func inspect(out io.Writer) bool {
	latestFeatures := getFeatures()
	atLeastOneFound := false

	for _, l := range latestFeatures {
		found := false
		for _, o := range latestFeatures {
			if o.Type == Outline {
				if o.Title == l.Title && dataRowSame(l.DataRow, o.DataRow) && getTestSuite(o) == getTestSuite(l) {
					found = true
				}
			} else {
				if o.Title == l.Title && getTestSuite(o) == getTestSuite(l) {
					found = true
				}
			}
			if found {
				if o.LineNumber != l.LineNumber {
					atLeastOneFound = true
					_, _ = io.WriteString(out, "-> Found Scenarios with same title on same file\n")
					_, _ = io.WriteString(out, fmt.Sprintf("\tScenario 1: %s\n", getTestPath(o)))
					_, _ = io.WriteString(out, fmt.Sprintf("\tScenario 2: %s\n", getTestPath(l)))
				}
				break
			}
		}
	}

	return atLeastOneFound
}

func checkDuplicates(out io.Writer) int {
	hasDuplicates := inspect(out)
	_, _ = io.WriteString(out, "\n")
	if hasDuplicates {
		_, _ = io.WriteString(out, "\n")
		_, _ = io.WriteString(out, "Duplicate scenarios found\n")
		return 1
	}
	return 0
}

type update struct {
	token      string
	linenumber int
}

func getUpdatesFromSteps(steps []object.Step) []update {
	var lastToken token.Type
	updates := []update{}
	for _, s := range steps {
		if lastToken == token.GIVEN || lastToken == token.WHEN || lastToken == token.THEN {
			if lastToken == s.Token.Type {
				updates = append(updates, update{
					s.Token.Type.String(),
					s.Token.LineNumber,
				})
			}
		}
		if s.Token.Type != token.AND {
			lastToken = s.Token.Type
		}
	}
	return updates
}

func checkAnd(out io.Writer) int {
	latestFeatures := getFeatures()

	var lastFeaturePath string
	for _, feature := range latestFeatures {
		updates := []update{}

		updates = append(updates, getUpdatesFromSteps(feature.Scenario.Steps)...)
		if lastFeaturePath != feature.FilePath && feature.Background != nil {
			updates = append(updates, getUpdatesFromSteps(feature.Background.Steps)...)
		}

		input, err := ioutil.ReadFile(feature.FilePath)
		if err != nil {
			_, _ = io.WriteString(out, err.Error())
			_, _ = io.WriteString(out, "\n")
			return 1
		}

		content := strings.Split(string(input), "\n")
		for _, u := range updates {
			_, _ = io.WriteString(out, fmt.Sprintf("Replacing %s from %s -> \"And\"\n\n", getTestPath(feature), u.token))
			content[u.linenumber-1] = strings.Replace(content[u.linenumber-1], u.token, "And", 1) // strings.Join(replaceString, " ")
		}

		err = ioutil.WriteFile(feature.FilePath, []byte(strings.Join(content, "\n")), 0644)
		if err != nil {
			_, _ = io.WriteString(out, err.Error())
			_, _ = io.WriteString(out, "\n")
			return 1
		}
		lastFeaturePath = feature.FilePath
	}
	return 0
}

func main() {
	out := new(bytes.Buffer)
	exitStatus := 0
	if len(os.Args) < 2 {
		help(out)
		exitStatus++
	} else {
		switch os.Args[1] {
		case "help":
			exitStatus += help(out)
		case "cache":
			exitStatus += checkDuplicates(out)
			exitStatus += cacheFeaturesData(out)
		case "shift":
			exitStatus += checkDuplicates(out)
			exitStatus += getShifts(out)
		case "inspect":
			exitStatus += checkDuplicates(out)
		case "check-and":
			exitStatus += checkAnd(out)
		case "scan":
			exitStatus += scanForNewScenarios(out)
			exitStatus += scanForRemovedScenarios(out)
		default:
			_, _ = io.WriteString(out, "opps! seems like you forgot to provide the path of the feature file")
			exitStatus++
		}
	}
	_, _= io.Copy(os.Stdout, out)
	os.Exit(exitStatus)
}

func dataRowSame(d1, d2 []object.TableData) bool {
	if len(d1) != len(d2) {
		return false
	}
	for i := range d1 {
		if d1[i].Literal != d2[i].Literal {
			return false
		}
	}

	return true
}

func getFeatures() []Feature {
	featuresPath := os.Getenv("FEATURES_PATH")

	if featuresPath == "" {
		log.Fatal(fmt.Errorf("error: setup features path with FEATURES_PATH env variable"))
	}

	path := featuresPath
	abs, err := filepath.Abs(path)
	if err != nil {
		log.Fatal(fmt.Errorf("error: invalid path provided, make sure the path %q is correct", path))
	}

	fi, err := os.Stat(abs)
	if os.IsNotExist(err) {
		log.Fatal(fmt.Errorf("error: make sure the path %q exists", abs))
	}

	var features []Feature

	switch mode := fi.Mode(); {
	case mode.IsDir():
		err := filepath.Walk(abs, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			var fileFeatures []Feature
			if !info.IsDir() {
				if ext := filepath.Ext(info.Name()); ext == ".feature" {
					fileFeatures = getFeaturesFromFile(path)
				}
			}
			features = append(features, fileFeatures...)

			return nil
		})
		if err != nil {
			log.Println(err)
		}
	case mode.IsRegular():
		features = getFeaturesFromFile(abs)
	}

	return features
}

func cacheFeaturesData(out io.Writer) int {
	features := getFeatures()

	bolB, _ := json.Marshal(features)

	err := ioutil.WriteFile("output.json", bolB, 0644)

	if err != nil {
		_, _ = io.WriteString(out, err.Error())
		_, _ = io.WriteString(out, "\n")
		return 1
	}
	return 0
}

func getFeaturesFromFile(filePath string) []Feature {
	var features []Feature
	l := lexer.NewFromFile(filePath)
	p := parser.New(l)
	res := p.Parse()
	if len(p.Errors()) != 0 {
		for _, err := range p.Errors() {
			panic(err)
		}
	} else {
		for _, feature := range res.Features {
			for _, scenario := range feature.Scenarios {
				outlineObj, ok := scenario.(*object.ScenarioOutline)
				isOutline := ok
				if isOutline {
					fullTables := object.Table{}
					for _, table := range outlineObj.Tables {
						fullTables.Append(table)
					}
					for i, s := range outlineObj.GetScenarios() {
						features = append(features, Feature{Outline, s.LineNumber, s.ScenarioText, fullTables[i+1], filePath, &s, feature.Background})
					}
				} else {
					s, _ := scenario.(*object.Scenario)
					features = append(features, Feature{Normal, s.LineNumber, s.ScenarioText, []object.TableData{}, filePath, s, feature.Background})
				}
			}
		}
	}
	return features
}

func getTestPath(feature Feature) string {
	dir := filepath.Dir(feature.FilePath)
	base := filepath.Base(dir)
	filename := filepath.Base(feature.FilePath)
	return fmt.Sprintf("%s:%d", filepath.Join(base, filename), feature.LineNumber)
}

func getTestSuite(feature Feature) string {
	dir := filepath.Dir(feature.FilePath)
	base := filepath.Base(dir)
	filename := filepath.Base(feature.FilePath)
	return filepath.Join(base, filename)
}

func getGithubLinkPath(path string) string {
	parts := strings.Split(path, ":")
	if len(parts) != 2 {
		log.Fatal("Could not parse path")
	}
	return fmt.Sprintf("%s#L%s", parts[0], parts[1])
}

func replaceOccurrenceInFile(shifts []shift, filePath string) error {
	input, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	newContents := string(input)
	for _, shift := range shifts {
		newContents = strings.Replace(
			newContents,
			fmt.Sprintf("[%s]", shift.oldPath),
			// adding extra line to avoid changing already changed line
			fmt.Sprintf("[%s]", shift.newPath+"आफ्नै बादल"),
			-1,
		)
	}

	for _, shift := range shifts {
		newContents = strings.Replace(
			newContents,
			fmt.Sprintf("%s)", getGithubLinkPath(shift.oldPath)),
			// adding extra line to avoid changing already changed line
			fmt.Sprintf("%s%s)", getGithubLinkPath(shift.newPath), "आफ्नै बादल"),
			-1,
		)
	}

	// Remove added extra line
	newContents = strings.Replace(newContents, "आफ्नै बादल", "", -1)

	err = ioutil.WriteFile(filePath, []byte(newContents), 0644)
	if err != nil {
		return err
	}
	return nil
}

// scanForNewScenarios scans for new scenarios added after caching
func scanForNewScenarios(out io.Writer) int {
	latestFeatures := getFeatures()
	oldFeaturesData, err := ioutil.ReadFile("output.json")
	if err != nil {
		_, _ = io.WriteString(out, err.Error())
		return 1
	}

	var oldFeatures []Feature
	_ = json.Unmarshal(oldFeaturesData, &oldFeatures)

	for _, l := range latestFeatures {
		found := false
		for _, o := range oldFeatures {
			if o.Type == Outline {
				if o.Title == l.Title && dataRowSame(o.DataRow, l.DataRow) && getTestSuite(o) == getTestSuite(l) {
					found = true
				}
			} else {
				if o.Title == l.Title && getTestSuite(o) == getTestSuite(l) {
					found = true
				}
			}
		}
		if !found {
			_, _ = io.WriteString(out, "found new scenario\n")
			_, _ = io.WriteString(out, "New: "+getTestPath(l))
			_, _ = io.WriteString(out, "\n\n")
		}
	}
	return 0
}

// scanForRemovedScenarios scans for old scenarios removed after caching
func scanForRemovedScenarios(out io.Writer) int {
	latestFeatures := getFeatures()
	oldFeaturesData, err := ioutil.ReadFile("output.json")
	if err != nil {
		_, _ = io.WriteString(out, err.Error())
		return 1
	}

	var oldFeatures []Feature
	_ = json.Unmarshal(oldFeaturesData, &oldFeatures)

	for _, o := range oldFeatures {
		found := false
		for _, l := range latestFeatures {
			if o.Type == Outline {
				if o.Title == l.Title && dataRowSame(o.DataRow, l.DataRow) && getTestSuite(o) == getTestSuite(l) {
					found = true
				}
			} else {
				if o.Title == l.Title && getTestSuite(o) == getTestSuite(l) {
					found = true
				}
			}
		}
		if !found {
			_, _ = io.WriteString(out, "scenario got removed\n")
			_, _ = io.WriteString(out, "Deleted: "+getTestPath(o))
			_, _ = io.WriteString(out, "\n\n")
		}
	}
	return 0
}

func help(out io.Writer) int {
	_, _ = io.WriteString(out, "ocBddKit " + version + ", A tool to manage feature files for ownCloud\n")
	_, _ = io.WriteString(out, "Usage: ocBddKit <option>\n\n")
	_, _ = io.WriteString(out, "Available Options:\n")
	_, _ = io.WriteString(out, "\thelp    - to display this help message\n")
	_, _ = io.WriteString(out, "\tinspect - to inspect the feature file/dir if acceptable for caching\n")
	_, _ = io.WriteString(out, "\tcache   - to cache a feature file/dir\n")
	_, _ = io.WriteString(out, "\tshift   - to update the expected failures files\n")
	_, _ = io.WriteString(out, "\tscan    - to scan if new scenarios were added or old scenarios were deleted\n")
	_, _ = io.WriteString(out, "\n")

	_, _ = io.WriteString(out, "Instructions:\n")
	_, _ = io.WriteString(out, "\t Use the existing commitID in .drone.env from respective projects to 'inspect' and 'cache'\n")
	_, _ = io.WriteString(out, "\t Then, checkout to the latest version of that project.\n")
	_, _ = io.WriteString(out, "\t After that, update the expected failures files with 'shift' command or 'scan' for new or removed scenarios\n")
	return 0
}
