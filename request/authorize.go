package request

import (
	"encoding/json"
	"fmt"

	"github.com/Connect-Club/connectclub-mobile-common/storage"
	"github.com/Connect-Club/connectclub-mobile-common/utils"
	log "github.com/sirupsen/logrus"
)

func (h *HttpClientStruct) Authorize(clientQuery string) string {
	query := fmt.Sprintf("client_id=%s&client_secret=%s", clientId, clientSecret)
	if len(clientQuery) > 0 {
		query = fmt.Sprintf("%s&%s", query, clientQuery)
	}

	endpoint := fmt.Sprintf("%s/oauth/v2/token", h.params.endpoint)
	response := h.makeRequest(requestParams{
		endpoint:           endpoint,
		method:             "GET",
		useAuthorizeHeader: false,
		query:              query,
	})
	jsonMap := utils.H{}
	err := json.Unmarshal([]byte(response), &jsonMap)
	if err != nil {
		return errorReadBody
	}

	statusCode := jsonMap["code"].(float64)
	if isNotSuccess(statusCode) {
		return response
	}

	body := jsonMap["body"].(map[string]interface{})
	if body["error"] != nil {
		return response
	}

	access := body["access_token"].(string)
	refresh := body["refresh_token"].(string)
	if len(access) > 0 && len(refresh) > 0 {
		storage.Get().SetString("accessToken", access)
		storage.Get().SetString("refreshToken", refresh)
		h.params.refreshCountCall = 0
		log.Infof("🪕 set accessToken: %v", access)
		return emptySuccess
	}
	return response
}
