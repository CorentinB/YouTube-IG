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
	"github.com/labstack/gommon/color"
)

var checkPre = color.Yellow("[") + color.Green("✓") + color.Yellow("]") + color.Yellow("[")

func appendID(ID string, worker *sync.WaitGroup, client *redis.Client) {
	defer worker.Done()
	exist, err := client.SIsMember("ids", ID).Result()
	if exist == false {
		err = client.SAdd("ids", ID, 0).Err()
		if err != nil {
			log.Fatal(err)
		} else {
			fmt.Println(checkPre + color.Cyan(ID) + color.Yellow("]") + color.Green(" Added to the DB!"))
		}
	}
}

func grabSuggest(ID string, client *redis.Client) {
	// start workers group
	var wg sync.WaitGroup
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
			wg.Add(1)
			go appendID(videoID, &wg, client)
			wg.Wait()
		}
	})
}

func processSingleID(ID string, worker *sync.WaitGroup, client *redis.Client) {
	defer worker.Done()
	grabSuggest(ID, client)
}

func processList(maxConc int64, client *redis.Client, worker *sync.WaitGroup) {
	defer worker.Done()
	var count int64

	// start workers group
	var wg sync.WaitGroup
	var randomID string

	// scan the list line by line
	for {
		count++
		nbElements, err := client.SCard("ids").Result()
		if err != nil {
			log.Fatal(err)
		}
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
		go processSingleID(randomID, &wg, client)
		if count == maxConc {
			wg.Wait()
			count = 0
		}
	}

	wg.Wait()
	processList(maxConc, client, worker)
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

	// Start workers group
	var wg sync.WaitGroup
	var maxConc int64
	maxConc = 16
	wg.Add(1)
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
	go processList(maxConc, client, &wg)
	wg.Wait()
}

func main() {
	start := time.Now()
	argumentParsing(os.Args[1:])
	color.Println(color.Cyan("Done in ") + color.Yellow(time.Since(start)) + color.Cyan("!"))
}
