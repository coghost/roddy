package roddy

import (
	"hash/fnv"
	"io"
	"net/url"
	"regexp"

	"github.com/coghost/xutil"
)

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

// FilenameFromUrl takes input as a escaped url & outputs filename from it (unescaped - normal one)
func FilenameFromUrl(urlstr string) (string, error) {
	u, err := url.Parse(urlstr)
	if err != nil {
		return "", err
	}

	return xutil.RefineString(u.Scheme + "_" + u.Host), nil
}

func isMatchingFilter(fs []*regexp.Regexp, d []byte) bool {
	for _, r := range fs {
		if r.Match(d) {
			return true
		}
	}

	return false
}

func normalizeURL(u string) string {
	parsed, err := urlParser.Parse(u)
	if err != nil {
		return u
	}

	return parsed.String()
}

func requestHash(url string, body io.Reader) uint64 {
	h := fnv.New64a()
	// reparse the url to fix ambiguities such as
	// "http://example.com" vs "http://example.com/"
	io.WriteString(h, normalizeURL(url))

	if body != nil {
		io.Copy(h, body)
	}

	return h.Sum64()
}
