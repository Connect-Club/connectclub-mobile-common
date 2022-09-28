package utils

import (
	"encoding/json"
	"log"
)

type H map[string]interface{}

func Unmarshal(data []byte) H {
	jsonMap := H{}
	_ = json.Unmarshal(data, &jsonMap)
	return jsonMap
}

func PackToJsonString(jsonMap H) string {
	data, err := json.Marshal(jsonMap)
	if err != nil {
		log.Printf("Error marshal message %v", jsonMap)
		return "{}"
	}
	return string(data)
}

func PackToByteArray(jsonMap interface{}) []byte {
	data, err := json.Marshal(jsonMap)
	if err != nil {
		log.Printf("Error marshal message %v", jsonMap)
		return make([]byte, 0)
	}
	return data
}

