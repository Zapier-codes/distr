package buildtrigger

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// ErrAssetNotFound means GitHub does not have the asset, or the token cannot see it.
var ErrAssetNotFound = errors.New("the release asset was not found")

// ResolveAssetDownload asks GitHub for the short-lived signed URL of a release asset, with the token of this
// instance, and returns it without following it. The asset of a private repository cannot be fetched without that
// token, so the user's browser is sent to the signed URL instead of to GitHub with a credential. Only the
// identifiers are used to build the request, never anything a requester typed.
func (c *Client) ResolveAssetDownload(ctx context.Context, repository string, assetID int64) (string, error) {
	// repository was validated as owner/name when it was stored and is compared to the configured one on the way in.
	endpoint := fmt.Sprintf("%v/repos/%v/releases/assets/%v", c.endpoint, repository, assetID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("could not create asset request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.config.Token)
	req.Header.Set("Accept", "application/octet-stream")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	// The redirect is the answer, so it must not be followed: the signed URL is for the user's browser.
	noRedirect := *c.client
	noRedirect.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := noRedirect.Do(req)
	if err != nil {
		return "", fmt.Errorf("asset request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusFound, http.StatusMovedPermanently, http.StatusSeeOther, http.StatusTemporaryRedirect:
		location := resp.Header.Get("Location")
		if err := checkAssetLocation(location); err != nil {
			return "", err
		}
		return location, nil
	case http.StatusNotFound:
		return "", ErrAssetNotFound
	default:
		return "", fmt.Errorf("GitHub answered the asset request with status %v%v",
			resp.StatusCode, githubMessage(resp.Body))
	}
}

// checkAssetLocation accepts only an https URL on a GitHub download host. The caller redirects a browser there, so
// whatever answered the request must not be able to send a user anywhere else.
func checkAssetLocation(location string) error {
	u, err := url.Parse(location)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return errors.New("GitHub answered with a download URL that is not https")
	}
	host := strings.ToLower(u.Hostname())
	if host != "github.com" && !strings.HasSuffix(host, ".github.com") && !strings.HasSuffix(host, ".githubusercontent.com") {
		return errors.New("GitHub answered with a download URL on an unexpected host")
	}
	return nil
}
