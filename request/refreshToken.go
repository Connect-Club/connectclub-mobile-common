package request

import (
	"encoding/json"
	"fmt"
	"github.com/Connect-Club/connectclub-mobile-common/storage"
	"github.com/Connect-Club/connectclub-mobile-common/utils"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
)

func (h HttpClientStruct) refreshToken() bool {
	h.params.refreshCountCall += 1
	refreshToken := storage.Get().GetString("refreshToken")
	log.Info("🪕 %v", refreshToken)
	if len(refreshToken) == 0 {
		return false
	}
	query := fmt.Sprintf("grant_type=refresh_token&client_id=%s&client_secret=%s&refresh_token=%s", clientId, clientSecret, refreshToken)
	endpoint := fmt.Sprintf("%s/oauth/v2/token", h.params.endpoint)
	url := fmt.Sprintf("%s?%s", endpoint, query)
	log.Infof("🪕 request %v", url)
	resp, err := h.params.client.Get(url)
	if err != nil {
		log.WithError(err).Error("http request error")
		return false
	}
	defer func() {
		//just in case, empty the response body
		_, _ = io.Copy(ioutil.Discard, resp.Body)
		if err := resp.Body.Close(); err != nil {
			log.WithError(err).Warn("close body error")
		}
	}()
	log.Infof("🪕 response %v", resp.StatusCode)

	storage.Get().Delete("accessToken")
	storage.Get().Delete("refreshToken")

	if !success(resp) {
		return false
	}

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithError(err).Error("read body error")
		return false
	}
	jsonMap := utils.H{}
	err = json.Unmarshal(responseBody, &jsonMap)
	if err != nil {
		log.WithError(err).Error("unmarshal response body error")
	}
	log.Infof("🪕 response body=%v", jsonMap)

	if jsonMap["error"] != nil {
		return false
	}

	access := jsonMap["access_token"].(string)
	refresh := jsonMap["refresh_token"].(string)
	if len(access) > 0 && len(refresh) > 0 {
		storage.Get().SetString("accessToken", access)
		storage.Get().SetString("refreshToken", refresh)
		log.Infof("🪕 set accessToken %v", access)
		return true
	}
	return false
}
