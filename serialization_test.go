package main

import (
	"io"
	"os"
	"testing"
)

func openTestFile(fileName string) (io.ReadCloser, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func TestVideo(t *testing.T) {
	reader, err := openTestFile("api-2020-01-21.json")
	defer reader.Close()
	if err != nil {
		t.Error(err)
	}
	var apiPicture picture
	err = makePictureFromAPI(reader, &apiPicture)
	if err != nil {
		t.Error(err)
	}

	htmlReader, err := openTestFile("ap200121.html")
	defer htmlReader.Close()
	if err != nil {
		t.Error(err)
	}
	var htmlPicture picture
	err = makePictureFromHTML(htmlReader, &htmlPicture)
	if err != nil {
		t.Error(err)
	}

	if apiPicture != htmlPicture {
		t.Errorf("\n%v\nis not equal to\n%v", apiPicture, htmlPicture)
	}
}

func TestImage(t *testing.T) {
	reader, err := openTestFile("api-2020-01-28.json")
	defer reader.Close()
	if err != nil {
		t.Error(err)
	}
	var apiPicture picture
	err = makePictureFromAPI(reader, &apiPicture)
	if err != nil {
		t.Error(err)
	}

	htmlReader, err := openTestFile("ap200128.html")
	defer htmlReader.Close()
	if err != nil {
		t.Error(err)
	}
	var htmlPicture picture
	err = makePictureFromHTML(htmlReader, &htmlPicture)
	if err != nil {
		t.Error(err)
	}

	// HTML doesn't support copyright for now
	apiPicture.Copyright = ""
	if apiPicture != htmlPicture {
		t.Errorf("\n%v\nis not equal to\n%v", apiPicture, htmlPicture)
	}
}
