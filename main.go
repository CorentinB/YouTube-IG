package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/labstack/gommon/color"
	"github.com/remeh/sizedwaitgroup"
)

var checkPre = color.Yellow("[") + color.Green("✓") + color.Yellow("]") + color.Yellow("[")

// GetIDs is for https://youtube.the-eye.eu/api/admin/requests?secret=SECRET&limit=10000
type GetIDs struct {
	Ok       bool   `json:"ok"`
	Msg      string `json:"msg"`
	Requests []struct {
		ID         int         `json:"ID"`
		VideoID    string      `json:"video_id"`
		RawURL     string      `json:"raw_url"`
		ArchivedAt interface{} `json:"archived_at"`
	} `json:"requests"`
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
	rand.Seed(time.Now().Unix())
	IDs := new(GetIDs)

	getJSON("https://youtube.the-eye.eu/api/admin/requests?secret="+arguments.Secret+"&limit=10000", IDs)

	n := rand.Int() % len(IDs.Requests)

	return IDs.Requests[n].VideoID
}

func pushIDs(videoIDs []string) {
	data := new(Payload)
	data.VideoIds = videoIDs
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		// handle err
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", "https://ytma.frenchy.space/api/admin/requests", body)
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