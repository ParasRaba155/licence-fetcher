package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"time"
)

const (
	// npmURL should be used with package name
	npmURL = "https://registry.npmjs.org/%s"
	// githubURL should be used with owner and repo args
	githubURL = "https://api.github.com/repos/%s/%s"
)

const defaultTimeout = 10 * time.Second

var gitHubURLRegexp = regexp.MustCompile(`.+\/\/github\.com\/([A-Za-z0-9\-_]+)\/([A-Za-z0-9\-_]+)\.git`)
var errNotAGithubURL = errors.New("not a github url")

// fetchGithubURLforNPMPackage for package pkg
func fetchGithubURLforNPMPackage(pkg string) (string, error) {
	url := fmt.Sprintf(npmURL, pkg)
	httpClient := http.Client{Timeout: defaultTimeout}

	req, err := http.NewRequest("GET", url, http.NoBody)
	if err != nil {
		slog.Error("fetchGithubURLforNPMPackage: request create", SlogErrorAttr(err))
		return "", err
	}

	jsonResp, err := HandleHTTPRequest(req, httpClient)
	if err != nil {
		return "", err
	}

	var resp NpmRegistryResp
	err = json.Unmarshal(jsonResp, &resp)
	if err != nil {
		slog.Error("fetchGithubURLforNPMPackage: response Unmarshal", SlogErrorAttr(err))
		return "", err
	}
	return resp.GetGitURL(), nil
}

func parseGithubURL(url string) (owner, repo string, err error) {
	parts := gitHubURLRegexp.FindStringSubmatch(url)
	if len(parts) != 3 {
		return "", "", fmt.Errorf("%w: %q", errNotAGithubURL, url)
	}
	return parts[1], parts[2], nil
}

func fetchLicenseFromGithubRepo(owner, repo string) (GithubRepoInfo, error) {
	var (
		url        = fmt.Sprintf(githubURL, owner, repo)
		httpClient = http.Client{Timeout: defaultTimeout}
		resp       GithubRepoInfo
	)

	req, err := http.NewRequest("GET", url, http.NoBody)
	if err != nil {
		slog.Error("fetchLicenseFromGithubRepo: request create", SlogErrorAttr(err))
		return resp, err
	}

	jsonResp, err := HandleHTTPRequest(req, httpClient)
	if err != nil {
		return resp, err
	}

	err = json.Unmarshal(jsonResp, &resp)
	if err != nil {
		slog.Error("fetchLicenseFromGithubRepo: response Unmarshal", SlogErrorAttr(err))
		return resp, err
	}
	return resp, nil
}

func main() {
	url, err := fetchGithubURLforNPMPackage("@babel/core")
	if err != nil {
		return
	}
	owner, repo, err := parseGithubURL(url)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	lic, err := fetchLicenseFromGithubRepo(owner, repo)
	fmt.Printf("err: %v, lic: %+v\n", err, lic)
}
