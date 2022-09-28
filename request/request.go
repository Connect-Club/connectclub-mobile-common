package request

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type HttpClient interface {
	Authorize(clientQuery string) string
	GetRequest(
		endpoint string,
		method string,
		useAuthorizeHeader bool,
		generateJwt bool,
		query string,
		body string,
		filePartName string,
		fileName string,
		filePath string,
	) string
	SendLogFileWithBodyText(string) string
	SendLogFileWithPath(path string, bodyText string) string
}

type FileParams struct {
	path     string
	partName string
}

type Params struct {
	userAgent        string
	endpoint         string
	client           *http.Client
	refreshCountCall int
}
type HttpClientStruct struct {
	params *Params
}

func New(
	endpoint string,
	platform string,
	version interface{},
	versionName interface{},
	buildNumber interface{},
) HttpClient {
	log.Infof("++ %v %v %v %v %v", endpoint, platform, version, versionName, buildNumber)
	return &HttpClientStruct{
		params: &Params{
			userAgent: fmt.Sprintf("%s %s/app %s (%s)", platform, version, versionName, buildNumber),
			endpoint:  endpoint,
			client:    &http.Client{},
		},
	}
}

func (h *HttpClientStruct) GetRequest(
	endpoint string,
	method string,
	useAuthorizeHeader bool,
	generateJwt bool,
	query string,
	body string,
	filePartName string,
	fileName string,
	filePath string,
) string {
	var fileParam *requestFile = nil
	if len(filePath) > 0 {
		if len(fileName) == 0 {
			fileName = "photo"
		}
		if len(filePartName) == 0 {
			filePartName = "photo"
		}
		fileParam = &requestFile{
			part: filePartName,
			path: filePath,
			name: fileName,
		}
	}
	var bodyParam *requestBody = nil
	if len(body) > 0 {
		bodyParam = &requestBody{
			part: "",
			data: body,
		}
	}
	result := h.makeRequest(requestParams{
		endpoint:           endpoint,
		method:             method,
		useAuthorizeHeader: useAuthorizeHeader,
		generateJwt:        generateJwt,
		query:              query,
		body:               bodyParam,
		file:               fileParam,
	})
	return result
}
