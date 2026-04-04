package appmeta

// AppVersion is the product version displayed in UI and exposed via /api/app/meta.
// Keep in sync with release documentation and CHANGELOG.md.
var AppVersion = "1.0.1"

const (
	ProductName = "Berkut Solutions - TARINIO"

	// RepositoryURL is intentionally kept generic unless a dedicated public repo exists.
	RepositoryURL = "https://github.com/BerkutSolutions"

	// GitHubAPIReleases is used by update-check logic (if enabled).
	// If you publish releases in a dedicated repo, update both RepositoryURL and this constant.
	GitHubAPIReleases = ""
)

