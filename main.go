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
}

type shift struct {
	oldPath string
	newPath string
}

func getShifts() {
	latestFeatures := getFeatures()

	dat, err := ioutil.ReadFile("output.json")
	if err != nil {
		panic(err)
	}

	oldFeatures := []Feature{}
	shifts := []shift{}

	json.Unmarshal(dat, &oldFeatures)

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
			err = replaceOccuranceInFile(shifts, filepath.Join(expectedFailuresDir, f.Name()))
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func inspect() bool {
	latestFeatures := getFeatures()
	atLeastOnefound := false

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
					atLeastOnefound = true
					fmt.Println("\nFound Scenarios with same title on same file")
					fmt.Println("Old :", getTestPath(o))
					fmt.Println("New: ", getTestPath(l))
				}
				break
			}
		}
	}

	return atLeastOnefound
}

func checkDuplicates() {
	hasDuplicates := inspect()
	fmt.Println()
	if hasDuplicates {
		log.Fatal(fmt.Errorf("Duplicate Scenarios found"))
		os.Exit(1)
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

func checkAnd() error {
	latestFeatures := getFeatures()

	for _, feature := range latestFeatures {
		type update struct {
			token      string
			linenumber int
		}
		updates := []update{}
		var lastToken token.Type
		for _, s := range feature.Scenario.Steps {
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
	}
	return nil
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal(fmt.Errorf("Opps, Seems like you forgot to provide the path of the feature file"))
		os.Exit(1)
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
	case "check_and":
		checkAnd()
	default:
		log.Fatal(fmt.Errorf("Opps, Seems like you forgot to provide the path of the feature file"))
		os.Exit(1)
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

func getFeatures() []Feature {
	featuresPath := os.Getenv("FEATURES_PATH")

	if featuresPath == "" {
		log.Fatal(fmt.Errorf("Setup features Path with FEATURES_PATH env variable"))
		os.Exit(1)
	}

	path := featuresPath
	abs, err := filepath.Abs(path)
	if err != nil {
		log.Fatal(fmt.Errorf("Invalid path provided, Make sure the path %q is correct", path))
		os.Exit(1)
	}

	fi, err := os.Stat(abs)
	if os.IsNotExist(err) {
		log.Fatal(fmt.Errorf("Error, Make sure the path %q exists", abs))
		os.Exit(1)
	}

	features := []Feature{}

	switch mode := fi.Mode(); {
	case mode.IsDir():
		err := filepath.Walk(abs, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			fileFeatures := []Feature{}
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
	features := []Feature{}
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

						features = append(features, Feature{Outline, s.LineNumber, s.ScenarioText, outlineObj.Table[i+1], filePath, &s})
					}
				} else {
					s, _ := scenario.(*object.Scenario)
					features = append(features, Feature{Normal, s.LineNumber, s.ScenarioText, []object.TableData{}, filePath, s})
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

func replaceOccuranceInFile(shifts []shift, filePath string) error {
	input, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	newContents := string(input)
	for _, shift := range shifts {
		newContents = strings.Replace(
			newContents,
			fmt.Sprintf("[%s]", shift.oldPath),
			// adding extra line to avoid changing alreaady chantged line
			fmt.Sprintf("[%s]", shift.newPath+"आफ्नै बादल"),
			-1,
		)
	}

	for _, shift := range shifts {
		newContents = strings.Replace(
			newContents,
			fmt.Sprintf("%s)", getGithubLinkPath(shift.oldPath)),
			// adding extra line to avoid changing alreaady chantged line
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
