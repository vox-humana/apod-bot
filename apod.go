package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	apodURL        = "https://apod.nasa.gov/apod.rss"
	apodPageURL    = "https://apod.nasa.gov/apod/ap%s.html"
	apodAPIURL     = "https://api.nasa.gov/planetary/apod?api_key=DEMO_KEY&date=%s"
	mediaTypeImage = "image"
	mediaTypeVideo = "video"
)

type picture struct {
	Copyright    string `json:"copyright"`
	Date         string `json:"date"`
	Explanation  string `json:"explanation"`
	Title        string `json:"title"`
	MediaType    string `json:"media_type"`
	FullImageURL string `json:"hdurl"`
	URL          string `json:"url"`
	Link         string
}

func readPicture(reader io.Reader) picture {
	body, err := ioutil.ReadAll(reader)
	if err != nil {
		logError(err)
	}

	var item picture
	err = json.Unmarshal(body, &item)
	if err != nil {
		logError(err)
	}

	item.removeAds()

	// Sometimes copyright contains new lines :shrug:
	item.Copyright = strings.ReplaceAll(item.Copyright, "\n", " ")
	return item
}

func (p *picture) removeAds() {
	adStartIndex := strings.Index(p.Explanation, "   ")
	if adStartIndex != -1 {
		p.Explanation = p.Explanation[0:adStartIndex]
	}
}

func checkResponseStatus(resp *http.Response) error {
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("Bad response status: %s %s", resp.Status, string(body))
	}
	return nil
}

// func firstWords(s string, count int) string {
// 	runes := []rune(s)
// 	for i := range runes {
// 		if unicode.IsSpace(runes[i]) {
// 			count--
// 			if count == 0 {
// 				return string(runes[0:i])
// 			}
// 		}
// 	}
// 	return s
// }

func firstSentences(s string, count int) string {
	for i := range s {
		c := s[i]
		if strings.ContainsAny(string(c), "?.!") {
			count--
			if count == 0 {
				return string(s[0:i])
			}
		}
	}
	return s
}

func pictureURL(p picture) string {
	const dateFormat = "060102"
	pictureTime, err := time.Parse("2006-01-02", p.Date)
	if err != nil {
		logError(err)
	}
	pictureDate := pictureTime.Format(dateFormat)
	if pictureDate != time.Now().Format(dateFormat) {
		logError("Picture's date doesn't match current date:", pictureDate)
	}
	return fmt.Sprintf(apodPageURL, pictureDate)
}

func makeRequest(currentDate string) io.ReadCloser {
	apiURL := fmt.Sprintf(apodAPIURL, currentDate)
	resp, err := http.Get(apiURL)
	if err != nil {
		logError(err)
	}

	fmt.Println("APOD API response:", resp.StatusCode)
	err = checkResponseStatus(resp)
	if err != nil {
		logError(err)
	}

	return resp.Body
}

func openTestFile(fileName string) io.ReadCloser {
	f, err := os.Open(fileName)
	if err != nil {
		logError(err)
	}
	return f
}

var sendError func(text string) error = func(string) error {
	return nil
}

func logError(v ...interface{}) {
	appname := filepath.Base(os.Args[0])
	text := fmt.Sprintln(v...)
	sendError("Error from `" + appname + "`: " + text)
	log.Fatalln(text)
}

func main() {
	var service, token string
	var chatID int64
	var err error
	var errChatID int64
	flag.StringVar(&token, "token", "", "bot api token")
	flag.Int64Var(&chatID, "chat", 0, "destination chat id")
	flag.StringVar(&service, "service", "tt", "tg or tt")
	flag.Int64Var(&errChatID, "err_chat", 0, "chat for error notification")
	flag.Parse()

	if len(token) == 0 || chatID == 0 {
		log.Fatalln("Wrong arguments")
	}

	if errChatID != 0 {
		if service == "tg" {
			sendError = func(s string) error {
				message := tgMessage{errChatID, s, ""}
				return tgSendMessage(message, tgSendMessageTemplate, token)
			}
		} else {
			sendError = func(s string) error {
				url := fmt.Sprintf(ttSendMessageTemplate, token, errChatID)
				return ttSendMessage(url, ttMessage{s, []ttMessageAttachment{}, true}, 0)
			}
		}
	}

	lastDate := readLastSentDate()
	fmt.Println("Last sent date:", lastDate)

	currentTime := time.Now()
	currentDate := currentTime.Format("2006-01-02")
	if lastDate == currentDate {
		fmt.Println("Nothing to do")
		return
	}

	reader := makeRequest(currentDate)
	//reader, _ := os.Open("api-2020-01-01.json")
	defer reader.Close()
	item := readPicture(reader)
	item.Link = pictureURL(item)

	var send func(picture, string, int64) error
	if service == "tg" {
		if item.MediaType == mediaTypeImage {
			send = tgSendPicture
		} else if item.MediaType == mediaTypeVideo {
			send = tgSendVideo
		} else {
			logError("Unsupported TG media_type", item.MediaType)
		}
	} else {
		if item.MediaType == mediaTypeImage {
			send = ttSendPicture
		} else if item.MediaType == mediaTypeVideo {
			send = ttSendVideo
		} else {
			logError("Unsupported TT media_type", item.MediaType)
		}
	}
	err = send(item, token, chatID)
	if err != nil {
		logError(strings.ToUpper(service), err)
	}

	saveCurrentDate(currentDate)
}
