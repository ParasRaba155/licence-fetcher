package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sync"
)

type dependencyInfo struct {
	name        string
	licenseInfo LicenseInfo
}

func ReadPackageJSON(rd io.Reader) ([]dependencyInfo, error) {
	// Read the package.json file
	jsonFile, readAllErr := io.ReadAll(rd)
	if readAllErr != nil {
		slog.Error("couldn't read package.json file", SlogErrorAttr(readAllErr))
		return nil, readAllErr
	}

	// Unmarshal the JSON data
	var pkgJSON PackageJSON
	if unmarshalErr := json.Unmarshal(jsonFile, &pkgJSON); unmarshalErr != nil {
		slog.Error("couldn't unmarshal package.json", SlogErrorAttr(unmarshalErr))
		return nil, unmarshalErr
	}

	if len(pkgJSON.Dependencies) == 0 {
		slog.Debug("no Dependencies in package.json", slog.Any("pkgJSON", pkgJSON))
	}

	// Channel to limit the number of concurrent goroutines
	const maxGoroutines = 5
	sem := make(chan struct{}, maxGoroutines)

	// A channel to collect results
	resultChan := make(chan dependencyInfo, len(pkgJSON.Dependencies))
	// A channel to collect errors
	errChan := make(chan error, len(pkgJSON.Dependencies))

	var wg sync.WaitGroup

	for dependency := range pkgJSON.Dependencies {
		wg.Add(1)
		sem <- struct{}{} // Acquire a slot in the goroutine pool

		go func(dependency string, wg *sync.WaitGroup) {
			defer func() {
				wg.Done()
				<-sem
			}() // Release the slot in the goroutine pool

			// Fetch the GitHub URL for the npm package
			githubURL, err := fetchGithubURLforNPMPackage(dependency)
			if err != nil {
				errChan <- err
				resultChan <- dependencyInfo{
					name:        dependency,
					licenseInfo: ErrorLicenseInfo("fetch Github URL"),
				}
				return
			}

			// Parse the GitHub URL to extract the owner and repo
			owner, repo, err := parseGithubURL(githubURL)
			if err != nil {
				errChan <- err
				resultChan <- dependencyInfo{
					name:        dependency,
					licenseInfo: ErrorLicenseInfo("parse Github URL"),
				}
				return
			}

			// Fetch license information from GitHub repo
			repoInfo, err := fetchLicenseFromGithubRepo(owner, repo)
			if err != nil {
				errChan <- err
				resultChan <- dependencyInfo{
					name:        dependency,
					licenseInfo: ErrorLicenseInfo("fetch License from github"),
				}
				return
			}

			// Send result to the result channel
			resultChan <- dependencyInfo{
				name:        dependency,
				licenseInfo: repoInfo.License,
			}
		}(dependency, &wg)
	}

	// Wait for all goroutines to finish
	go func() {
		wg.Wait()
		close(resultChan)
		close(errChan)
	}()

	// Collect results and check for errors
	result := []dependencyInfo{}
	for info := range resultChan {
		result = append(result, info)
	}

	// Handle any errors that occurred during processing
	select {
	case err := <-errChan:
		return result, err
	default:
		// No errors, return result
		return result, nil
	}
}

var pkgJSONContent = []byte(`
{
  "name": "typescript-orm-benchmark",
  "module": "index.ts",
  "type": "module",
  "devDependencies": {
    "@pgtyped/cli": "^2.3.0",
    "@types/pg": "^8.11.6",
    "@types/pg-pool": "^2.0.6",
    "bun-types": "latest"
  },
  "peerDependencies": {
    "typescript": "^5.0.0"
  },
  "dependencies": {
    "@faker-js/faker": "^8.4.1",
    "@mikro-orm/core": "^6.3.0",
    "@mikro-orm/mysql": "^6.3.0",
    "@mikro-orm/postgresql": "^6.3.0",
    "@pgtyped/runtime": "^2.3.0",
    "@prisma/client": "^5.17.0",
    "drizzle-orm": "^0.32.0",
    "knex": "^3.1.0",
    "kysely": "^0.27.4",
    "mariadb": "^3.3.1",
    "mitata": "^0.1.11",
    "mysql2": "^3.10.3",
    "pg": "^8.12.0",
    "pg-pool": "^3.6.2",
    "postgres": "~3.4.4",
    "prisma": "latest",
    "reflect-metadata": "^0.2.2",
    "sequelize": "^6.37.3",
    "ts-node": "^10.9.2",
    "typeorm": "^0.3.20"
  }
}
`)

func main() {
	rd := bytes.NewReader(pkgJSONContent)
	info, err := ReadPackageJSON(rd)
	fmt.Printf("err: %s\n", err)
	for _, temp := range info {
		fmt.Printf("<%s>: <%s>\n", temp.name, temp.licenseInfo.Key)
	}
}
