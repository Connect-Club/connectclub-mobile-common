package common

import (
	"crypto/tls"
	"github.com/Connect-Club/connectclub-mobile-common/request"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type PublicHttpClient interface {
	request.HttpClient
}

func InsecureHttpTransport() {
	log.Warn("using insecure TLS client")
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
}

func HttpClient(
	endpoint string,
	platform string,
	version string,
	versionName string,
	buildNumber string,
) PublicHttpClient {
	return request.New(endpoint, platform, version, versionName, buildNumber)
}
