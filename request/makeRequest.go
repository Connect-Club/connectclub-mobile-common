package request

import (
	"encoding/json"
	"fmt"
	jwt_generator "github.com/Connect-Club/connectclub-jwt-generator"
	"github.com/Connect-Club/connectclub-mobile-common/storage"
	"github.com/Connect-Club/connectclub-mobile-common/utils"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

const clientId = "1_3u3bpqxw736s4kgo0gsco4kw48gos800gscg4s4w8w80oogc8c"
const clientSecret = "6cja0geitwsok4gckw0cc0c04sc0sgwgo8kggcoc08wocsw8wg"

const unauthorized = "{\"error\":\"unauthorized\"}"
const loggedOut = "{\"error\":\"loggedOut\"}"
const errorReadBody = "{\"error\":\"errorReadBody\"}"
const emptySuccess = "{}"

type requestParams struct {
	endpoint           string
	method             string
	useAuthorizeHeader bool
	generateJwt        bool
	query              string
	body               *requestBody
	file               *requestFile
}

type requestFile struct {
	part string
	name string
	path string
}

type requestBody struct {
	part string
	data string
}

func (h *HttpClientStruct) makeRequest(params requestParams) string {
	amplSessionId := storage.Get().GetString("amplitudeSessionId")
	amplDeviceId := storage.Get().GetString("amplitudeDeviceId")
	accessToken := storage.Get().GetString("accessToken")
	isLogout := strings.Contains(params.endpoint, "account/logout")
	isEmptyToken := len(accessToken) == 0
	if h.params.refreshCountCall > 10 {
		return loggedOut
	}
	if isLogout && isEmptyToken {
		return loggedOut
	}
	if isEmptyToken && params.useAuthorizeHeader && isLogout {
		return loggedOut
	}
	if isEmptyToken && params.useAuthorizeHeader {
		return unauthorized
	}
	headers := map[string][]string{
		"Accept":       {"application/json"},
		"Content-Type": {"application/json"},
		"User-Agent":   {h.params.userAgent},
	}
	if len(amplSessionId) > 0 {
		headers["amplSessionId"] = []string{amplSessionId}
	}
	if len(amplDeviceId) > 0 {
		headers["amplDeviceId"] = []string{amplDeviceId}
	}
	if params.useAuthorizeHeader {
		log.Infof("🪕 set useAuthorizeHeader. accessToken=%v", accessToken)
		headers["Authorization"] = []string{fmt.Sprintf("Bearer %s", accessToken)}
	} else if params.generateJwt {
		jwt := jwt_generator.GenerateJwt()
		log.Infof("🪕 set generateJwt. jwt=%v", jwt)
		headers["Authorization"] = []string{fmt.Sprintf("Bearer %s", jwt)}
	}
	parsedUrl, _ := url.Parse(params.endpoint)
	parsedUrl.RawQuery = parseQuery(params.query)
	log.Infof("🪕 %v %v (%v) useAuthorizeHeader=%v amplSessionId=%v amplDeviceId=%v",
		params.method,
		parsedUrl,
		params.body,
		params.useAuthorizeHeader,
		headers["amplSessionId"],
		headers["amplDeviceId"],
	)
	request := &http.Request{
		Method: params.method,
		URL:    parsedUrl,
		Header: headers,
	}
	setBody(request, params.body, params.file)
	log.Info("beforeSend body")
	res, err := h.params.client.Do(request)
	if err != nil {
		log.WithError(err).Error("can not do request")
		code := -1
		if res != nil {
			code = res.StatusCode
		}
		return utils.PackToJsonString(utils.H{
			"code":  code,
			"error": "errorRequest",
		})
	}
	log.Infof("🪕 Request.response code=%v method=%v parsedUrl=%v", res.StatusCode, params.method, parsedUrl)
	if res.StatusCode == http.StatusUnauthorized {
		if !h.refreshToken() {
			return unauthorized
		}
		return h.makeRequest(params)
	}
	if strings.Contains(params.endpoint, "account/logout") {
		storage.Get().Delete("accessToken")
		storage.Get().Delete("refreshToken")
	}
	responseBody, bodyReadErr := ioutil.ReadAll(res.Body)
	if bodyReadErr != nil {
		return errorReadBody
	}
	log.Infof("🪕 response body=%v", string(responseBody))
	var body utils.H
	if err := json.Unmarshal(responseBody, &body); err != nil {
		return errorReadBody
	}
	if success(res) {
		return utils.PackToJsonString(utils.H{
			"code": res.StatusCode,
			"body": body,
		})
	}

	if body["errors"] != nil {
		errors := body["errors"].([]interface{})
		if len(errors) == 0 {
			return utils.PackToJsonString(utils.H{
				"code":  res.StatusCode,
				"error": "serverError",
				"body":  body,
			})
		}
		errorString := errors[0].(string)
		return utils.PackToJsonString(utils.H{
			"code":  res.StatusCode,
			"error": errorString,
			"body":  body,
		})
	}
	return utils.PackToJsonString(utils.H{
		"code":  res.StatusCode,
		"error": "serverError",
		"body":  body,
	})
}

func success(r *http.Response) bool {
	return r.StatusCode >= http.StatusOK && r.StatusCode < http.StatusMultipleChoices
}

func isNotSuccess(code float64) bool {
	return code < http.StatusOK || code >= http.StatusMultipleChoices
}
