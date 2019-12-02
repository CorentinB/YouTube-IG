package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"

	"github.com/PuerkitoBio/goquery"
	"github.com/labstack/gommon/color"
	"github.com/remeh/sizedwaitgroup"
)

var checkPre = color.Yellow("[") + color.Green("✓") + color.Yellow("]") + color.Yellow("[")

// Seeds is for https://youtube.the-eye.eu/api/admin/seed?secret=SECRET
type Seeds struct {
	Ok    bool     `json:"ok"`
	Msg   string   `json:"msg"`
	Seeds []string `json:"seeds"`
}

// Payload to push DIs
type Payload struct {
	VideoIds []string `json:"video_ids"`
}

func getJSON(url string, target interface{}) error {
	r, err := http.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(target)
}

func getRandomID() string {
	IDs := new(Seeds)

	getJSON("https://youtube.the-eye.eu/api/admin/seed?secret="+arguments.Secret, IDs)

	return IDs.Seeds[0]
}

func pushIDs(videoIDs []string) {
	data := new(Payload)
	data.VideoIds = videoIDs
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		// handle err
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", "https://youtube.the-eye.eu/api/admin/requests", body)
	if err != nil {
		// handle err
	}
	req.Header.Set("X-Secret", arguments.Secret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// handle err
	}
	defer resp.Body.Close()
}

func grabSuggestion(worker *sizedwaitgroup.SizedWaitGroup) {
	defer worker.Done()
	var videoIDs []string

	ID := getRandomID()

	req, err := http.NewRequest("GET", "http://youtube.com/watch?v="+ID+"&gl=US&hl=en&has_verified=1&bpctr=9999999999", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "http.request error for %s: %s", ID, err)
		runtime.Goexit()
	}
	req.Header.Set("Connection", "close")
	req.Close = true
	html, err := http.DefaultClient.Do(req)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		runtime.Goexit()
	}
	// check status, exit if != 200
	if html.StatusCode != 200 {
		fmt.Fprintf(os.Stderr, "Status code error for %s: %d %s", ID, html.StatusCode, html.Status)
		runtime.Goexit()
	}
	body, err := ioutil.ReadAll(html.Body)

	// start goquery in the page
	document, err := goquery.NewDocumentFromReader(bytes.NewReader(body))

	// extract suggested IDs
	document.Find("span").Each(func(i int, s *goquery.Selection) {
		if name, _ := s.Attr("class"); name == "yt-uix-simple-thumb-wrap yt-uix-simple-thumb-related" {
			videoID, _ := s.Attr("data-vid")
			fmt.Println(checkPre + color.Cyan(videoID) + color.Yellow("]") + color.Green(" Discovered"))
			videoIDs = append(videoIDs, videoID)
		}
	})

	pushIDs(videoIDs)
}

func crawl() {
	var worker = sizedwaitgroup.New(arguments.Concurrency)

	for {
		worker.Add()
		go grabSuggestion(&worker)
	}
}

func main() {
	// Parse arguments
	parseArgs(os.Args)

	crawl()
}
