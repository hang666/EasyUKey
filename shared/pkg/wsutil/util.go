package wsutil

import (
	"fmt"
	"net/url"
	"strings"
)

// ConvertHTTPToWS 将HTTP/HTTPS URL转换为WS/WSS URL
func ConvertHTTPToWS(httpURL string) (string, error) {
	u, err := url.Parse(httpURL)
	if err != nil {
		return "", err
	}

	if u.Scheme == "https" {
		u.Scheme = "wss"
	} else if u.Scheme == "http" {
		u.Scheme = "ws"
	} else {
		return "", fmt.Errorf("不支持的协议: %s", u.Scheme)
	}

	u.Path = strings.TrimSuffix(u.Path, "/")
	return u.String(), nil
}
