package main

import (
	"encoding/json"
	"fmt"
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

const (
	Outline scenarioType = "Outline"
	Normal               = "Normal"
)

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

func getShifts() {
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
					fmt.Println("\nFound Shift")
					fmt.Println("Old :", getTestPath(o))
					fmt.Println("New: ", getTestPath(l))

					shifts = append(shifts, shift{getTestPath(o), getTestPath(l)})
				}
				break
			}
		}

		if !found {
			fmt.Println("new scenario found")
			fmt.Println("New: ", getTestPath(l))
			fmt.Println("")
		}
	}

	expectedFailuresDir := os.Getenv("EXPECTED_FAILURES_DIR")
	expectedFailuresPrefix := os.Getenv("EXPECTED_FAILURES_PREFIX")

	if expectedFailuresPrefix == "" {
		expectedFailuresPrefix = "expected-failure"
	}

	files, err := ioutil.ReadDir(expectedFailuresDir)
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		if strings.HasPrefix(f.Name(), expectedFailuresPrefix) {
			err = replaceOccurrenceInFile(shifts, filepath.Join(expectedFailuresDir, f.Name()))
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func inspect() bool {
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
					fmt.Println("\n-> Found Scenarios with same title on same file")
					fmt.Println("\tScenario 1:", getTestPath(o))
					fmt.Println("\tScenario 2:", getTestPath(l))
				}
				break
			}
		}
	}

	return atLeastOneFound
}

func checkDuplicates() {
	hasDuplicates := inspect()
	fmt.Println()
	if hasDuplicates {
		log.Fatal(fmt.Errorf("opps! duplicate scenarios found"))
	}
}

func delete_empty(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
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
		if (s.Token.Type != token.AND) {
			lastToken = s.Token.Type
		}
	}
	return updates
}

func checkAnd() error {
	latestFeatures := getFeatures()

	var lastFeaturePath string
	for _, feature := range latestFeatures {
		updates := []update{}

		updates = append(updates, getUpdatesFromSteps(feature.Scenario.Steps)...)
		if (lastFeaturePath != feature.FilePath && feature.Background != nil) {
			updates = append(updates, getUpdatesFromSteps(feature.Background.Steps)...)
		}

		input, err := ioutil.ReadFile(feature.FilePath)
		if err != nil {
			return err
		}

		content := strings.Split(string(input), "\n")
		for _, u := range updates {
			fmt.Printf("Replacing %s:%d from %s -> \"And\"\n", feature.FilePath, u.linenumber, u.token)
			content[u.linenumber-1] = strings.Replace(content[u.linenumber-1], u.token, "And", 1) // strings.Join(replaceString, " ")
		}

		err = ioutil.WriteFile(feature.FilePath, []byte(strings.Join(content, "\n")), 0644)
		if err != nil {
			return err
		}
		lastFeaturePath = feature.FilePath
	}
	return nil
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal(fmt.Errorf("opps! seems like you forgot to provide the path of the feature file"))
	}

	switch os.Args[1] {
	case "cache":
		checkDuplicates()
		cacheFeaturesData()
	case "shift":
		checkDuplicates()
		getShifts()
	case "inspect":
		checkDuplicates()
	case "check-and":
		checkAnd()
	case "scan":
		scanForNewScenarios()
		scanForRemovedScenarios()
	default:
		log.Fatal(fmt.Errorf("opps! seems like you forgot to provide the path of the feature file"))
	}
}

func dataRowSame(d1, d2 []object.TableData) bool {
	if len(d1) != len(d2) {
		return false
	}
	for i, _ := range d1 {
		if d1[i].Literal != d2[i].Literal {
			return false
		}
	}

	return true
}

func getFeatures() ([]Feature) {
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

func cacheFeaturesData() {
	features := getFeatures()

	bolB, _ := json.Marshal(features)

	err := ioutil.WriteFile("output.json", bolB, 0644)

	if err != nil {
		panic(err)
	}
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
					for i, s := range outlineObj.GetScenarios() {

						features = append(features, Feature{Outline, s.LineNumber, s.ScenarioText, outlineObj.Table[i+1], filePath, &s, feature.Background})
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
		fmt.Println(parts)
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
			fmt.Sprintf("[%s]", shift.newPath + "आफ्नै बादल"),
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
