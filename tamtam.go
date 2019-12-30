package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"path"
	"time"
)

const (
	uploadTemplate        = "https://botapi.tamtam.chat/uploads?access_token=%s&type=%s"
	sendMessageTemplate   = "https://botapi.tamtam.chat/messages?access_token=%s&chat_id=%d"
	maxMessageSendRetries = 10
	fileAttachmentType    = "file"
	imageAttachmentType   = "image"
)

var (
	tamTamToken  string
	tamTamChatID int64
)

type message struct {
	Text        string              `json:"text"`
	Attachments []messageAttachment `json:"attachments"`
}

type messageAttachment struct {
	Type    string            `json:"type"`
	Payload attachmentPayload `json:"payload"`
}

type attachmentPayload struct {
	Token string `json:"token"`
}

func getUploadURL(attachmentType string) (string, error) {
	url := fmt.Sprintf(uploadTemplate, tamTamToken, attachmentType)
	req, err := http.Post(url, "application/json", nil)
	if err != nil {
		return "", err
	}
	defer req.Body.Close()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return "", err
	}
	fmt.Println("Get upload URL response body:", string(body))

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
	fmt.Println("Upload attachment response body:", string(body))

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
			fmt.Println("extracting from", value)
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
	return "", errors.New("Can't extract token" + string(body))
}

func uploadAttachment(remoteURL string, attachmentType string) (string, error) {
	uploadURL, err := getUploadURL(attachmentType)
	if err != nil {
		return "", err
	}

	token, err := uploadFile(remoteURL, uploadURL, attachmentType == imageAttachmentType)
	if err != nil {
		return "", err
	}
	return token, nil
}

func postMessage(url string, message interface{}, numberOfRetries int) {
	json, err := json.Marshal(message)
	if err != nil {
		log.Fatalln("Failed to create message JSON")
		return
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(json))
	if err != nil {
		log.Fatalln("Failed to send message")
	}
	defer resp.Body.Close()

	fmt.Printf("Post message (%d retries) response status: %d\n", numberOfRetries, resp.StatusCode)
	// retry on 400
	if resp.StatusCode == http.StatusBadRequest {
		if numberOfRetries == maxMessageSendRetries {
			fmt.Println("Max send retries exceed")
			return
		}
		time.Sleep(2 * time.Second)
		postMessage(url, message, numberOfRetries+1)
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("bad status: %s\n", resp.Status)
	}
	body, _ := ioutil.ReadAll(resp.Body)

	fmt.Println("Post message response body:", string(body))
}

func sendAPIPicture(picture apiPicture, link string) {
	fileToken, err := uploadAttachment(picture.FullImageURL, fileAttachmentType)
	if err != nil {
		log.Fatalln(err)
	}

	if len(fileToken) == 0 {
		log.Fatalln("empty upload file token")
	}

	imageToken, err := uploadAttachment(picture.ImageURL, imageAttachmentType)
	if err != nil {
		log.Fatalln(err)
	}

	if len(imageToken) == 0 {
		log.Fatalln("empty upload image token")
	}

	fileAttachment := messageAttachment{Type: fileAttachmentType, Payload: attachmentPayload{fileToken}}
	imageAttachment := messageAttachment{Type: imageAttachmentType, Payload: attachmentPayload{imageToken}}

	url := fmt.Sprintf(sendMessageTemplate, tamTamToken, tamTamChatID)

	text := picture.Title + "\n" + link

	postMessage(url, message{text, []messageAttachment{imageAttachment}}, 0)
	postMessage(url, message{"", []messageAttachment{fileAttachment}}, 0)
}
