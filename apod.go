package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	apodPageURL    = "https://apod.nasa.gov/apod/ap%s.html"
	apodAPIURL     = "https://api.nasa.gov/planetary/apod?api_key=DEMO_KEY&date=%s"
	apodSiteURL    = "https://apod.nasa.gov/apod/"
	mediaTypeImage = "image"
	mediaTypeVideo = "video"
)

func makeAPIRequest(currentTime time.Time) (io.ReadCloser, error) {
	currentDate := currentTime.Format("2006-01-02")
	apiURL := fmt.Sprintf(apodAPIURL, currentDate)
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, err
	}

	fmt.Println("APOD API response:", resp.StatusCode)
	err = checkResponseStatus(resp)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func makeHTMLRequest(currentTime time.Time) (io.ReadCloser, error) {
	currentDate := currentTime.Format("060102")
	url := fmt.Sprintf(apodPageURL, currentDate)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	fmt.Println("APOD HTML response:", resp.StatusCode)
	err = checkResponseStatus(resp)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func openTestFile(fileName string) (io.ReadCloser, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func pictureURL(p picture) string {
	const dateFormat = "060102"
	pictureTime, err := time.Parse("2006-01-02", p.Date)
	if err != nil {
		logError(err)
	}
	pictureDate := pictureTime.Format(dateFormat)
	if pictureDate != time.Now().Format(dateFormat) {
		logError("Picture's date doesn't match the current date:", pictureDate)
	}
	return fmt.Sprintf(apodPageURL, pictureDate)
}

var sendError func(text string) error = func(string) error {
	return nil
}

func logError(v ...interface{}) {
	appname := filepath.Base(os.Args[0])
	text := fmt.Sprintln(v...)
	sendError("❗️`" + appname + "`: " + text)
	log.Fatalln(text)
}

func logWarning(v ...interface{}) {
	appname := filepath.Base(os.Args[0])
	text := fmt.Sprintln(v...)
	sendError("⚠️`" + appname + "`: " + text)
	fmt.Println(text)
}

func pictureFromAPI(p *picture, t time.Time) error {
	reader, err := makeAPIRequest(t)
	if err != nil {
		logError(err)
	}
	defer reader.Close()
	return makePictureFromAPI(reader, p)
}

func pictureFromHTML(p *picture, t time.Time) error {
	reader, err := makeHTMLRequest(t)
	if err != nil {
		logError(err)
	}
	defer reader.Close()
	return makePictureFromHTML(reader, p)
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

	var item picture
	// try to get data from API first
	err = pictureFromAPI(&item, currentTime)
	if err != nil {
		logWarning("Got error from API", err)
		err = pictureFromHTML(&item, currentTime)
		if err != nil {
			logError(err)
		}
	}
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
