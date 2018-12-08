package main

import (
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
	"github.com/go-redis/redis"
	"github.com/gosuri/uilive"
	"github.com/labstack/gommon/color"
)

var checkPre = color.Yellow("[") + color.Green("✓") + color.Yellow("]")

var pipeLength = 0

var writer = uilive.New()

var added = 0

var processed = 0

var start = time.Now()

func appendID(ID string, pipe redis.Pipeliner) {
	processed++
	result, err := pipe.SAdd("ids", ID, 0).Result()
	if err != nil {
		log.Fatal(err)
	} else {
		if result == 1 {
			pipeLength++
			added++
		}
	}
	if pipeLength >= 1000 {
		_, err = pipe.Exec()
		if err != nil {
			log.Fatal(err)
		}
		pipeLength = 0
	}
}

func grabSuggest(ID string, pipe redis.Pipeliner, worker *sync.WaitGroup) {
	defer worker.Done()
	// request video html page

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
			go appendID(videoID, pipe)
		}
	})
}

func processList(maxConc int64, client *redis.Client) {
	var count int64

	// Start workers group
	var wg sync.WaitGroup
	var randomID string

	// Create Redis pipeline
	pipe := client.Pipeline()

	// Grab number of IDs
	nbElements, err := client.SCard("ids").Result()
	if err != nil {
		log.Fatal(err)
	}

	writer.Start()
	var averageTime, timeSince int64

	// Scan the list line by line
	for {
		count++
		if int(nbElements) > 0 {
			randomID, err = client.SRandMember("ids").Result()
			if err != nil {
				log.Fatal(err)
			}
		} else {
			err = client.SAdd("ids", "rYs84rkOgOg", 0).Err()
			if err != nil {
				log.Fatal(err)
			}
			randomID = "rYs84rkOgOg"
		}
		wg.Add(1)
		go grabSuggest(randomID, pipe, &wg)
		timeSince = int64(time.Since(start)) / 1e9
		if timeSince != 0 {
			averageTime = int64(processed) / (int64(time.Since(start)) / 1e9)
		}
		fmt.Fprintln(writer, checkPre+
			color.Yellow("[")+
			color.Cyan(processed)+
			color.Yellow(" IDs processed in ")+
			color.Cyan(time.Since(start))+
			color.Yellow("] ")+
			color.Cyan(added)+
			color.Green(" new IDs added to the DB! ")+
			color.Yellow("[Processing ")+
			color.Cyan(averageTime)+
			color.Cyan(" IDs")+
			color.Yellow("/")+
			color.Cyan("s")+
			color.Yellow("] "))

		if count == maxConc {
			wg.Wait()
			count = 0
		}
	}
}

func argumentParsing(args []string) {
	dbID, _ := strconv.ParseInt(args[3], 10, 64)
	// Connect to Redis server
	client := redis.NewClient(&redis.Options{
		Addr:     args[1],
		Password: args[2],
		DB:       int(dbID),
	})
	pong, err := client.Ping().Result()
	if pong != "PONG" {
		fmt.Println("Unable to connect to Redis DB")
		log.Fatal(err)
		os.Exit(1)
	}

	var maxConc int64
	maxConc = 16
	if len(args) > 4 {
		color.Red("Usage: ./youtube-ig [CONCURRENCY] [REDIS-HOST] [REDIS-PASSWORD] [REDIS-DB]")
		os.Exit(1)
	} else if len(args) == 4 {
		if _, err := strconv.ParseInt(args[0], 10, 64); err == nil {
			maxConc, _ = strconv.ParseInt(args[0], 10, 64)
		} else {
			color.Red("Usage: ./youtube-ig [CONCURRENCY] [REDIS-HOST] [REDIS-PASSWORD] [REDIS-DB]")
			os.Exit(1)
		}
	}
	processList(maxConc, client)
}

func main() {
	argumentParsing(os.Args[1:])
	color.Println(color.Cyan("Done in ") + color.Yellow(time.Since(start)) + color.Cyan("!"))
}
