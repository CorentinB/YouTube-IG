package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/labstack/gommon/color"
)

var checkPre = color.Yellow("[") + color.Green("✓") + color.Yellow("]") + color.Yellow("[")

func appendID(path string, ID string, file *os.File, worker *sync.WaitGroup) {
	defer worker.Done()
	found := 0
	// scan the list line by line
	scanner := bufio.NewScanner(file)
	// scan the list line by line
	for scanner.Scan() {
		if scanner.Text() == ID {
			found = 1
		}
	}
	// log if error
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	if found == 0 {
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		if _, err := f.Write([]byte(ID + "\n")); err != nil {
			log.Fatal(err)
		}
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
		fmt.Println(checkPre + color.Cyan(ID) + color.Yellow("]") + color.Green(" Added to the list!"))
	}
}

func grabSuggest(path string, ID string, file *os.File) {
	// start workers group
	var wg sync.WaitGroup
	// request video html page
	html, err := http.Get("http://youtube.com/watch?v=" + ID + "&gl=US&hl=en&has_verified=1&bpctr=9999999999")
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
			wg.Add(1)
			go appendID(path, videoID, file, &wg)
			wg.Wait()
		}
	})
}

func processSingleID(path string, ID string, worker *sync.WaitGroup, file *os.File) {
	defer worker.Done()
	grabSuggest(path, ID, file)
}

func processList(maxConc int64, path string, worker *sync.WaitGroup) {
	defer worker.Done()
	var count int64
	// start workers group
	var wg sync.WaitGroup
	// open file
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	// scan the list line by line
	scanner := bufio.NewScanner(file)
	// scan the list line by line
	for scanner.Scan() {
		count++
		wg.Add(1)
		go processSingleID(path, scanner.Text(), &wg, file)
		if count == maxConc {
			wg.Wait()
			count = 0
		}
	}
	// log if error
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	wg.Wait()
	processList(maxConc, path, worker)
}

func argumentParsing(args []string) {
	// start workers group
	var wg sync.WaitGroup
	var maxConc int64
	maxConc = 16
	wg.Add(1)
	if len(args) > 2 {
		color.Red("Usage: ./youtube-ig [list of IDs] [CONCURRENCY]")
		os.Exit(1)
	} else if len(args) == 2 {
		if _, err := strconv.ParseInt(args[1], 10, 64); err == nil {
			maxConc, _ = strconv.ParseInt(args[1], 10, 64)
		} else {
			color.Red("Usage: ./youtube-ig [list of IDs] [CONCURRENCY]")
			os.Exit(1)
		}
	}
	go processList(maxConc, args[0], &wg)
	wg.Wait()
}

func main() {
	start := time.Now()
	argumentParsing(os.Args[1:])
	color.Println(color.Cyan("Done in ") + color.Yellow(time.Since(start)) + color.Cyan("!"))
}
