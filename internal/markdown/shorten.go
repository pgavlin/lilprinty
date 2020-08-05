package markdown

import (
	"fmt"
	"net/url"

	"github.com/pgavlin/lilprinty/internal/util"
)

func shortenURL(urlString string) (string, error) {
	const baseURL = "http://tinyurl.com/api-create.php?url="

	parsed, err := url.Parse(urlString)
	if err != nil {
		return "", err
	}
	if !parsed.IsAbs() || parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", &url.Error{
			Op:  "parse",
			URL: urlString,
			Err: fmt.Errorf("only absolute HTTP and HTTPS URLs can be shortened"),
		}
	}

	contents, _, err := util.DownloadFile(baseURL + url.QueryEscape(parsed.String()))
	if err != nil {
		return "", err
	}
	return string(contents), nil
}
