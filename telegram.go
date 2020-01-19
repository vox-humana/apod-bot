package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	tgSendMessageTemplate = "https://api.telegram.org/bot%s/sendMessage?parse_mode=Markdown"
	tgSendPhotoTemplate   = "https://api.telegram.org/bot%s/sendPhoto?parse_mode=Markdown"
	tgSendFileTemplate    = "https://api.telegram.org/bot%s/sendDocument?parse_mode=Markdown"
)

type tgMessage struct {
	Chat int64  `json:"chat_id"`
	Text string `json:"text"`
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
		return fmt.Errorf("Bad response status: %s", resp.Status)
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
	document := tgDocumentMessage{chat, documentCaption, picture.FullImageURL, true}
	err = tgSendMessage(document, tgSendFileTemplate, token)
	if err != nil {
		return err
	}

	return nil
}

func tgSendVideo(picture picture, token string, chat int64) error {
	text := "[" + picture.Title + "](" + picture.URL + ")\n" + picture.Explanation
	message := tgMessage{chat, text}
	return tgSendMessage(message, tgSendMessageTemplate, token)
}
