## owncloud expected failure files updater
This is a tool that helps to update the expected failures file when you make changes on the feature files

### Usage
To use this tool run
```
go run main.go <command>
```

### Available commands
- #### Inspect
    Check for any duplicate scenarios in your feature files

    Required Env variables 

    - `FEATURES_PATH` Path where the Features files are


- #### Cache
    Read the Feature files and store their information

    Required Env variables 

    - `FEATURES_PATH` Path where the Features files are

- #### Shift
    Update the expected Failure files

    Required Env variables 

    - `FEATURES_PATH` Path where the Features files are
    - `EXPECTED_FAILURES_DIR` Path were expected failures files are
    - `EXPECTED_FAILURES_PREFIX` Prefix of the expected failure files in expectged failures dir (defaults to `expected-failure`)

### Instructions
- First check the .drone.env of the respective project to see the last version of testrunner used.
- Checkout to that version in the testrunner repo and cache the feature data with
    ```
    FEATURES_PATH=<path_to_feature_files> go run main.go cache
    ```
- Now checkout to the latest version in the testrunner repo and rerun the command with
    ```
    FEATURES_PATH=<path_to_feature_files> EXPECTED_FAILURES_DIR=<path_to_expected_failures> go run main.go shift
    ```

