SHELL=/bin/bash

.PHONY: help
help:
	@echo "Please use 'make <target>' where <target> is one of the following:"
	@echo
	@echo -e "\tinspect: to inspect the feature file/dir if acceptable for caching"
	@echo -e "\tcache: to cache a feature file/dir"
	@echo -e "\tshift: to update the expected failures files"
	@echo
	@echo -e "Instructions:"
	@echo
	@echo -e "\t- Use the existing commitID in .drone.env from respective projects to 'inspect' and 'cache'"
	@echo -e "\t- Then, checkout to the latest version of that project."
	@echo -e "\t- After that, update the expected failures files with 'shift' command or 'scan' for new or removed scenarios"

.PHONY: inspect
inspect: go.mod
	FEATURES_PATH=$$FEATURES_PATH go run main.go sacn.go inspect

.PHONY: cache
cache: go.mod
	FEATURES_PATH=$$FEATURES_PATH go run main.go scan.go cache

.PHONY: scan
scan: go.mod
	FEATURES_PATH=$$FEATURES_PATH go run main.go scan.go scan

.PHONY: shift
shift: go.mod
	FEATURES_PATH=$$FEATURES_PATH EXPECTED_FAILURES_DIR=$$EXPECTED_FAILURES_DIR EXPECTED_FAILURES_PREFIX=$$EXPECTED_FAILURES_PREFIX go run main.go scan.go shift
