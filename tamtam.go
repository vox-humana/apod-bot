package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"path"
	"time"
)

const (
	ttUploadTemplate        = "https://botapi.tamtam.chat/uploads?access_token=%s&type=%s"
	ttSendMessageTemplate   = "https://botapi.tamtam.chat/messages?access_token=%s&chat_id=%d"
	ttMaxMessageSendRetries = 10
	ttMessageSendRetryDelay = 2
	ttFileAttachmentType    = "file"
	ttImageAttachmentType   = "image"
)

type message struct {
	Text        string              `json:"text"`
	Attachments []messageAttachment `json:"attachments"`
	Notify      bool                `json:"notify"`
}

type messageAttachment struct {
	Type    string            `json:"type"`
	Payload attachmentPayload `json:"payload"`
}

type attachmentPayload struct {
	Token string `json:"token"`
}

func createUploadURL(attachmentType string, token string) (string, error) {
	url := fmt.Sprintf(ttUploadTemplate, token, attachmentType)
	req, err := http.Post(url, "application/json", nil)
	if err != nil {
		return "", err
	}
	defer req.Body.Close()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return "", err
	}
	fmt.Println("TT: Get upload URL response body:", string(body))

	var response struct {
		URL string `json:"url"`
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}

	return response.URL, nil
}

func uploadFile(sourceURL string, destinationURL string, isImage bool) (string, error) {
	resp, err := http.Get(sourceURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	_, filename := path.Split(sourceURL)

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormFile("data", filename)
	if err != nil {
		return "", err
	}

	_, err = io.Copy(fw, resp.Body)
	w.Close()
	if err != nil {
		return "", err
	}

	uploadResp, err := http.Post(destinationURL, w.FormDataContentType(), &b)
	if err != nil {
		return "", err
	}

	defer uploadResp.Body.Close()
	body, err := ioutil.ReadAll(uploadResp.Body)
	fmt.Println("TT: Upload attachment response body:", string(body))

	var response interface{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}

	if isImage {
		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		if err != nil {
			return "", err
		}
		photos := response["photos"].(map[string]interface{})
		for _, value := range photos {
			values := value.(map[string]interface{})
			token, prs := values["token"]
			if prs {
				return token.(string), nil
			}
		}
	} else {
		var response struct {
			Token string `json:"token"`
		}
		err = json.Unmarshal(body, &response)
		if err != nil {
			return "", err
		}
		return response.Token, nil
	}
	return "", errors.New("Can't extract token from" + string(body))
}

func uploadAttachment(remoteURL string, attachmentType string, token string) (string, error) {
	uploadURL, err := createUploadURL(attachmentType, token)
	if err != nil {
		return "", err
	}

	uploadToken, err := uploadFile(remoteURL, uploadURL, attachmentType == ttImageAttachmentType)
	if err != nil {
		return "", err
	}
	return uploadToken, nil
}

func ttSendMessage(url string, message interface{}, numberOfRetries int) error {
	json, err := json.Marshal(message)
	if err != nil {
		return errors.New("Failed to create message JSON")
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(json))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fmt.Printf("TT: Post message (%d retries) response status: %d\n", numberOfRetries, resp.StatusCode)

	if resp.StatusCode == http.StatusBadRequest {
		if numberOfRetries == ttMaxMessageSendRetries {
			return errors.New("Max send retries exceed")
		}
		time.Sleep(ttMessageSendRetryDelay * time.Second)
		return ttSendMessage(url, message, numberOfRetries+1)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Bad response status: %s", resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Println("TT: Post message response body:", string(body))
	return nil
}

func ttSendPicture(picture picture, token string, chat int64) error {
	fileToken, err := uploadAttachment(picture.FullImageURL, ttFileAttachmentType, token)
	if err != nil {
		return err
	}

	if len(fileToken) == 0 {
		return errors.New("Empty upload file token")
	}

	imageToken, err := uploadAttachment(picture.ImageURL, ttImageAttachmentType, token)
	if err != nil {
		return err
	}

	if len(imageToken) == 0 {
		return errors.New("Empty upload image token")
	}

	imageAttachment := messageAttachment{Type: ttImageAttachmentType, Payload: attachmentPayload{imageToken}}
	fileAttachment := messageAttachment{Type: ttFileAttachmentType, Payload: attachmentPayload{fileToken}}

	url := fmt.Sprintf(ttSendMessageTemplate, token, chat)

	text := "🌌" + picture.Title + "\n\n" + picture.Explanation + "\n🔗" + picture.Link

	err = ttSendMessage(url, message{text, []messageAttachment{imageAttachment}, true}, 0)
	if err != nil {
		return err
	}

	fileCaption := ""
	if len(picture.Copyright) > 0 {
		fileCaption = "© " + picture.Copyright
	}
	err = ttSendMessage(url, message{fileCaption, []messageAttachment{fileAttachment}, false}, 0)
	if err != nil {
		return err
	}

	return nil
}
