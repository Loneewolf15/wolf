package packager

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// RegistryResponse is the expected JSON response from the Wolf registry.
type RegistryResponse struct {
	Name       string   `json:"name"`
	Repository string   `json:"repository"`
	Latest     string   `json:"latest"`
	Versions   []string `json:"versions"`
}

var RegistryURL = "https://registry.wolf-lang.org/pkg"
var httpClient = &http.Client{Timeout: 10 * time.Second}

// ResolvePackage attempts to resolve a package name into a Git repository and version list.
func ResolvePackage(pkgName string) (*RegistryResponse, error) {
	// Direct Git hosting bypasses the vanity registry
	if strings.HasPrefix(pkgName, "github.com/") || strings.HasPrefix(pkgName, "gitlab.com/") || strings.HasPrefix(pkgName, "bitbucket.org/") {
		return &RegistryResponse{
			Name:       pkgName,
			Repository: "https://" + pkgName,
			// For direct hostings, we don't know the versions ahead of time 
			// without `git ls-remote`. The fetcher will handle it.
			Versions:   []string{},
		}, nil
	}

	url := fmt.Sprintf("%s/%s", RegistryURL, pkgName)
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("registry network error for %s: %w", pkgName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("package not found in registry: %s", pkgName)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("registry returned status %d for %s", resp.StatusCode, pkgName)
	}

	var regResp RegistryResponse
	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		return nil, fmt.Errorf("invalid registry response for %s: %w", pkgName, err)
	}

	return &regResp, nil
}
