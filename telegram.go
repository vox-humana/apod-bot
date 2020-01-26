package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"path"
	"strconv"
)

const (
	tgSendMessageTemplate = "https://api.telegram.org/bot%s/sendMessage"
	tgSendPhotoTemplate   = "https://api.telegram.org/bot%s/sendPhoto?parse_mode=Markdown"
	tgSendFileTemplate    = "https://api.telegram.org/bot%s/sendDocument?parse_mode=Markdown"
	tgParseModeMarkdown   = "Markdown"
	tgParseModeHTML       = "HTML"
)

type tgMessage struct {
	Chat      int64  `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

type tgPhotoMessage struct {
	Chat     int64  `json:"chat_id"`
	Text     string `json:"caption"`
	ImageURL string `json:"photo"`
}

type tgDocumentMessage struct {
	Chat    int64  `json:"chat_id"`
	Text    string `json:"caption"`
	FileURL string `json:"document"`
	Silent  bool   `json:"disable_notification"`
}

func fillForm(b *bytes.Buffer, chatID int64, caption string, remoteFileURL string) (string, error) {
	w := multipart.NewWriter(b)
	defer w.Close()

	// Fields
	fw, err := w.CreateFormField("chat_id")
	if err != nil {
		return "", err
	}
	io.WriteString(fw, strconv.FormatInt(chatID, 10))

	fw, err = w.CreateFormField("caption")
	if err != nil {
		return "", err
	}
	io.WriteString(fw, caption)

	fw, err = w.CreateFormField("disable_notification")
	if err != nil {
		return "", err
	}
	io.WriteString(fw, "true")

	// File
	resp, err := http.Get(remoteFileURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	_, filename := path.Split(remoteFileURL)

	fw, err = w.CreateFormFile("document", filename)
	if err != nil {
		return "", err
	}

	_, err = io.Copy(fw, resp.Body)

	if err != nil {
		return "", err
	}

	return w.FormDataContentType(), nil
}

// Even though sending file just by providing remoteURL exists,
// looks like it is more reliable to use multi-form POST
func tgSendDocument(chatID int64, caption string, remoteFileURL string, token string) error {
	var b bytes.Buffer
	ct, err := fillForm(&b, chatID, caption, remoteFileURL)
	if err != nil {
		return err
	}

	url := fmt.Sprintf(tgSendFileTemplate, token)
	resp, err := http.Post(url, ct, &b)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	fmt.Println("TG: POST document response body:", string(body))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Bad response status: %s (%s)", resp.Status, string(body))
	}
	return nil
}

func tgSendMessage(message interface{}, urlTemplate string, token string) error {
	json, err := json.Marshal(message)
	if err != nil {
		return err
	}

	url := fmt.Sprintf(urlTemplate, token)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(json))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fmt.Println("TG: Post message response status:", resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Bad response status: %s (%s)", resp.Status, string(body))
	}
	fmt.Println("TG: Post message response body:", string(body))

	return nil
}

func tgSendPicture(picture picture, token string, chat int64) error {
	explanation := firstSentences(picture.Explanation, 2) // TODO: max 1024
	photoCaption := "*" + picture.Title + "*\n" + explanation + "…\n" + picture.Link

	// Somehow TG sometimes doesn't like full image URLs (too big?)
	photo := tgPhotoMessage{chat, photoCaption, picture.URL}
	err := tgSendMessage(photo, tgSendPhotoTemplate, token)
	if err != nil {
		return err
	}

	documentCaption := ""
	if len(picture.Copyright) > 0 {
		documentCaption = "© " + picture.Copyright
	}
	// document := tgDocumentMessage{chat, documentCaption, picture.FullImageURL, true}
	// err = tgSendMessage(document, tgSendFileTemplate, token)
	return tgSendDocument(chat, documentCaption, picture.FullImageURL, token)
}

func tgSendVideo(picture picture, token string, chat int64) error {
	text := "[" + picture.Title + "](" + picture.URL + ")\n" + picture.Explanation
	message := tgMessage{chat, text, tgParseModeMarkdown}
	return tgSendMessage(message, tgSendMessageTemplate, token)
}
