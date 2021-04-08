package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

/**
 * scans for new scenarios added after caching
 */
func scanForNewScenarios() {
	latestFeatures := getFeatures()
	oldFeaturesData, err := ioutil.ReadFile("output.json")
	if err != nil {
		panic(err)
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
			fmt.Println("found new scenario")
			fmt.Println("New: ", getTestPath(l))
			fmt.Println("")
		}
	}
}

/**
 * scans for old scenarios removed after caching
 */
func scanForRemovedScenarios() {
	latestFeatures := getFeatures()
	oldFeaturesData, err := ioutil.ReadFile("output.json")
	if err != nil {
		panic(err)
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
			fmt.Println("scenario got removed")
			fmt.Println("Deleted: ", getTestPath(o))
			fmt.Println("")
		}
	}
}
