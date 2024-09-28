package main

type NpmRegistryResp struct {
	Versions map[string]struct {
		Repository struct {
			Type string `json:"type"`
			URL  string `json:"url"`
		} `json:"repository"`
	} `json:"versions"`
}

func (n NpmRegistryResp) GetGitURL() string {
	for _, version := range n.Versions {
		return version.Repository.URL
	}
	return ""
}

type GithubRepoInfo struct {
	License LicenseInfo `json:"license"`
}

type LicenseInfo struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	URL  string `json:"url"`
}
