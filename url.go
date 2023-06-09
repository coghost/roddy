package roddy

import "net/url"

// JoinUrlWithRef joins a URI reference from a base URL
func JoinUrlWithRef(baseUrl, refUrl string) (*url.URL, error) {
	u, err := url.Parse(refUrl)
	if err != nil {
		return nil, err
	}

	base, err := url.Parse(baseUrl)
	if err != nil {
		return nil, err
	}

	return base.ResolveReference(u), nil
}
