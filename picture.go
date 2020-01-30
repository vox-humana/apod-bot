package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/antchfx/htmlquery"
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

func makePictureFromHTML(reader io.Reader, p *picture) error {
	doc, err := htmlquery.Parse(reader)
	if err != nil {
		return err
	}
	titleNode, err := htmlquery.Query(doc, "//html/body/center[2]/b[1]")
	if err != nil {
		return err
	}
	title := htmlquery.InnerText(titleNode)

	explanationNode, err := htmlquery.Query(doc, "//html/body/p[1]")
	if err != nil {
		return err
	}
	explanation := htmlquery.InnerText(explanationNode)
	explanation = strings.Replace(explanation, "Explanation:", "", 1)

	imageNode, err := htmlquery.Query(doc, "//html/body/center[1]/p[2]/a/img")
	fullImageURL := ""
	imageURL := ""
	mediaType := mediaTypeImage
	if imageNode == nil {
		// try video
		imageNode, err = htmlquery.Query(doc, "//html/body/center[1]/p[2]/iframe")
		if imageNode == nil {
			return err
		}
		mediaType = mediaTypeVideo
		imageURL = htmlquery.SelectAttr(imageNode, "src")
	} else {
		imageURL = apodSiteURL + htmlquery.SelectAttr(imageNode, "src")
		fullImageNode, err := htmlquery.Query(doc, "//html/body/center[1]/p[2]/a")
		if err != nil {
			return err
		}
		fullImageURL = apodSiteURL + htmlquery.SelectAttr(fullImageNode, "href")
	}

	dateNode, err := htmlquery.Query(doc, "//html/body/center[1]/p[2]")
	dateText := trimSpaces(htmlquery.InnerText(dateNode))
	pictureTime, err := time.Parse("2006 January 2", dateText)
	if err != nil {
		return err
	}
	pictureDate := pictureTime.Format("2006-01-02")

	p.Title = title
	p.Explanation = explanation
	p.URL = imageURL
	p.FullImageURL = fullImageURL
	p.MediaType = mediaType
	p.Date = pictureDate
	p.trim()
	return nil
}

func makePictureFromAPI(reader io.Reader, p *picture) error {
	body, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, p)
	if err != nil {
		return err
	}

	p.removeAds()
	p.trim()
	return nil
}

func (p *picture) removeAds() {
	adStartIndex := strings.Index(p.Explanation, "   ")
	if adStartIndex != -1 {
		p.Explanation = p.Explanation[0:adStartIndex]
	}
}

func (p *picture) trim() {
	p.Title = trimSpaces(p.Title)
	p.Explanation = trimSpaces(p.Explanation)
	// Sometimes copyright contains new lines :shrug:
	p.Copyright = strings.ReplaceAll(p.Copyright, "\n", " ")
}

func checkResponseStatus(resp *http.Response) error {
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("Bad response status: %s %s", resp.Status, string(body))
	}
	return nil
}

func trimSpaces(s string) string {
	spaces := regexp.MustCompile(`\s+`)
	oneline := spaces.ReplaceAllString(s, " ")
	return strings.TrimSpace(oneline)
}
