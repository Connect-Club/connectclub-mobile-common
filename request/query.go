package request

import (
	"fmt"
	"net/url"
	"strings"
)

func parseQuery(query string) string {
	joinedQuery := ""
	queryCount := len(query)
	if queryCount > 0 {
		queryString := make([]string, 0)

		for _, element := range strings.Split(query, "&") {
			parts := strings.Split(element, "=")
			value := parts[1]
			if strings.EqualFold(value, "null") || strings.EqualFold(value, "undefined") {
				continue
			}
			if !strings.HasPrefix(value, "!") {
				value = url.QueryEscape(value)
			} else {
				value = strings.TrimPrefix(value, "!")
			}
			queryString = append(queryString, fmt.Sprintf("%s=%s", parts[0], value))
		}
		joinedQuery = strings.Join(queryString, "&")
	}
	return joinedQuery
}
