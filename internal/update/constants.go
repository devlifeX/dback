package update

const (
	GitHubOwner = "devlifeX"
	GitHubRepo  = "dback"
)

func GitHubRepoSlug() string {
	return GitHubOwner + "/" + GitHubRepo
}

var latestReleaseAPIURL = "https://api.github.com/repos/" + GitHubRepoSlug() + "/releases/latest"

func LatestReleaseAPIURL() string {
	return latestReleaseAPIURL
}
