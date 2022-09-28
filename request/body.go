package request

import (
	"bytes"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
)

func setBody(request *http.Request, requestBody *requestBody, requestFile *requestFile) {
	hasBody := requestBody != nil
	hasMultipartFile := requestFile != nil && len(requestFile.path) > 0
	// Nothing to do
	if !hasBody && !hasMultipartFile {
		return
	}
	hasMultipartBody := requestBody != nil && len(requestBody.part) > 0

	// Plaintext body only
	if hasBody && !hasMultipartBody && !hasMultipartFile {
		request.Body = ioutil.NopCloser(strings.NewReader(requestBody.data))
		return
	}

	bodyBuffer := &bytes.Buffer{}
	writer := multipart.NewWriter(bodyBuffer)

	if hasMultipartFile {
		file, err := os.Open(requestFile.path)
		if err != nil {
			log.WithError(err).Panic("can not open file")
		}
		defer file.Close()

		part, err := writer.CreateFormFile(requestFile.part, requestFile.name)
		if err != nil {
			log.WithError(err).Panic("can not create writer from file")
		}
		io.Copy(part, file)
		log.Infof("!!!! File part %v name %v type %v", requestFile.part, file.Name(), request.Header.Get("Content-Type"))
	}
	if hasMultipartBody {
		err := writer.WriteField(requestBody.part, requestBody.data)
		if err != nil {
			log.WithError(err).Error("set body error")
			return
		}
		log.Infof("!!!! Body part %v type %v", requestBody.part, request.Header.Get("Content-Type"))
	}
	err := writer.Close()
	if err != nil {
		log.WithError(err).Panic("can not close writer")
	}
	request.Body = ioutil.NopCloser(bodyBuffer)
	request.Header.Set("Content-Type", writer.FormDataContentType())
}
