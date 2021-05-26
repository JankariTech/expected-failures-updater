#!/usr/bin/env bash
# echo "# ""${lines[0]}" >&3

function setup() {
	# save hero feature content for future tests
	mkdir tests/tmp
	cp tests/fixtures/features/superHeroes/hero.feature tests/tmp/hero.feature

	# cache the existing feature
	export FEATURES_PATH=tests/fixtures/features
	go run main.go cache
}


@test "detect removed scenarios" {
	# remove a scenario from the cached feature
	sed -i '27,29d' tests/fixtures/features/superHeroes/hero.feature

	# scan the cached feature directory
	run go run main.go scan
	[ "$status" -eq 0 ]
	[ "${lines[1]}" = "scenario got removed" ]
	[ "${lines[2]}" = "Deleted:  superHeroes/hero.feature:27" ]
}

@test "detect new added scenarios" {
	# add a new scenario into the cached feature
	echo -e "\n  Scenario: new scene\n    When this\n    Then that\n" >> tests/fixtures/features/superHeroes/hero.feature

	# scan the cached feature directory
	run go run main.go scan

	[ "$status" -eq 0 ]
	[ "${lines[1]}" = "found new scenario" ]
	[ "${lines[2]}" = "New:  superHeroes/hero.feature:31" ]
}

function teardown() {
	# revert the modified feature file
	mv tests/tmp/hero.feature tests/fixtures/features/superHeroes/hero.feature
	rm -rf tests/tmp
}
