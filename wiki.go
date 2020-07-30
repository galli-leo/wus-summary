package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type ParseResult struct {
	Title     string
	Pageid    int64
	Text      map[string]string
	ParseTree map[string]string
}

type WikiResult struct {
	Parse ParseResult
}

type ImageInfo struct {
	Url string
}

type QueryPage struct {
	ImageInfo []ImageInfo
}

type QueryResult struct {
	Pages map[string]QueryPage
}

type ImageResult struct {
	Query QueryResult
}

var client = &http.Client{}
var wlog = newLogger("wiki")

func GetPage(page string, prop string) (*WikiResult, error) {
	log := wlog.With("page", page)
	log.Infof("Requesting wikipedia page: %v", page)

	// Create request
	req, err := http.NewRequest("GET", "https://en.wikipedia.org/w/api.php?action=parse&format=json&prop="+prop+"&page="+page, nil)

	if err != nil {
		log.Errorf("Failed to create request for page: %v", err)
		return nil, err
	}
	// Fetch Request
	resp, err := client.Do(req)
	log = log.With("status", resp.StatusCode)
	if err != nil {
		log.Errorf("Failed to request page: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		log.Errorf("Status code is not ok!")
		return nil, err
	}

	// data, err := ioutil.ReadAll(resp.Body)
	// log.Infof("Response data: %v", string(data))

	dec := json.NewDecoder(resp.Body)

	res := &WikiResult{}
	err = dec.Decode(res)
	if err != nil {
		log.Errorf("Failed to decode response: %v", err)
		return nil, err
	}

	return res, nil
}

func GetDistrLinks() []*Section {
	page, err := GetPage("List_of_probability_distributions", "text")
	if err != nil {
		wlog.Errorf("Failed to fetch list of probability distributions: %v", err)
		return []*Section{}
	}

	txt, ok := page.Parse.Text["*"]
	if !ok {
		wlog.Errorf("Failed to get parsed text from page: %v", page)
		return []*Section{}
	}

	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(txt))

	return ParseDistrLinks(doc)
}

func GetDistr(page string) *Distribution {
	res, err := GetPage(page, "parsetree")
	if err != nil {
		wlog.Errorf("Failed to fetch page: %v", err)
		return nil
	}

	txt, ok := res.Parse.ParseTree["*"]
	if !ok {
		wlog.Errorf("Failed to get parsed tree from page: %v", page)
		return nil
	}

	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(txt))

	if res.Parse.Title == "Beta distribution" {
		wlog.Infof("Kill me now daddy pls")
	}

	distr := ParseDistribution(doc)
	if distr != nil {
		distr.Name = res.Parse.Title
	}
	return distr
}

func GetImageUrl(filename string) string {
	log := wlog.With("filename", filename)
	log.Infof("Requesting wikipedia image")
	// Create client
	client := &http.Client{}

	// Create request
	req, err := http.NewRequest("GET", "https://en.wikipedia.org/w/api.php?action=query&titles=File:"+url.QueryEscape(filename)+"&prop=imageinfo&iiprop=timestamp%7Cuser%7Curl&format=json", nil)

	if err != nil {
		log.Errorf("Failed to create request for page: %v", err)
		return ""
	}
	// Fetch Request
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Failed to request page: %v", err)
		return ""
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)

	res := &ImageResult{}
	err = dec.Decode(res)
	if err != nil {
		log.Errorf("Failed to decode response: %v", err)
		return ""
	}

	for _, val := range res.Query.Pages {
		return val.ImageInfo[0].Url
	}

	return ""
}
