package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

const (
	apodURL     = "https://apod.nasa.gov/apod.rss"
	apodPageURL = "https://apod.nasa.gov/apod/ap%s.html"
	apodAPIURL  = "https://api.nasa.gov/planetary/apod?api_key=DEMO_KEY&date=%s"
)

type apiPicture struct {
	Copyright    string `json:"copyright"`
	Date         string `json:"date"`
	Explanation  string `json:"explanation"`
	FullImageURL string `json:"hdurl"`
	Title        string `json:"title"`
	ImageURL     string `json:"url"`
}

func main() {
	flag.StringVar(&tamTamToken, "token", "", "bot api token")
	flag.Int64Var(&tamTamChatID, "chat", 0, "destination chat id")
	flag.Parse()

	if len(tamTamToken) == 0 || tamTamChatID == 0 {
		log.Fatalln("Wrong arguments")
	}

	lastDate := readLastSentDate()
	fmt.Println("Last sent date: ", lastDate)

	time := time.Now()
	currentDate := time.Format("2006-01-02")
	if lastDate == currentDate {
		fmt.Println("Nothing to do")
		return
	}

	apiURL := fmt.Sprintf(apodAPIURL, currentDate)
	resp, err := http.Get(apiURL)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	var picture apiPicture
	err = json.Unmarshal(body, &picture)
	if err != nil {
		log.Fatalln(err)
	}

	shortDate := time.Format("060102")
	pageURL := fmt.Sprintf(apodPageURL, shortDate)
	sendAPIPicture(picture, pageURL)
	saveCurrentDate(currentDate)
}
