package request

import (
	"encoding/json"
	"fmt"
	"github.com/Connect-Club/connectclub-mobile-common/logs"
	"github.com/Connect-Club/connectclub-mobile-common/utils"
)

func (h *HttpClientStruct) SendLogFileWithBodyText(text string) string {
	logFilePath := logs.GetLogFilePath()
	if len(logFilePath) == 0 {
		return ""
	}
	return h.SendLogFileWithPath(logFilePath, text)
}

func (h *HttpClientStruct) SendLogFileWithPath(logFilePath string, bodyText string) string {
	endpoint := fmt.Sprintf("%s/v1/mobile-app-log", h.params.endpoint)
	bodyData := fmt.Sprintf("Log from GO %s", bodyText)
	response := h.makeRequest(
		requestParams{
			endpoint:           endpoint,
			method:             "POST",
			useAuthorizeHeader: true,
			query:              "",
			body: &requestBody{
				part: "body",
				data: bodyData,
			},
			file: &requestFile{
				part: "file",
				name: "log.txt",
				path: logFilePath,
			},
		},
	)
	jsonMap := utils.H{}
	err := json.Unmarshal([]byte(response), &jsonMap)
	if err != nil {
		return errorReadBody
	}

	return response
}
