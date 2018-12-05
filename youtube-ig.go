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
	"github.com/go-redis/redis"
	"github.com/labstack/gommon/color"
)

var checkPre = color.Yellow("[") + color.Green("✓") + color.Yellow("]") + color.Yellow("[")

func appendID(path string, ID string, file *os.File, worker *sync.WaitGroup, client *redis.Client) {
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
		exist, err := client.SIsMember("ids", ID).Result()
		if exist == false {
			err = client.SAdd("ids", ID, 0).Err()
			if err != nil {
				log.Fatal(err)
			} else {
				fmt.Println(checkPre + color.Cyan(ID) + color.Yellow("]") + color.Green(" Added to the DB!"))
			}
		}
		f.Close()
	}
}

func grabSuggest(path string, ID string, file *os.File, client *redis.Client) {
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
			go appendID(path, videoID, file, &wg, client)
			wg.Wait()
		}
	})
}

func processSingleID(path string, ID string, worker *sync.WaitGroup, file *os.File, client *redis.Client) {
	defer worker.Done()
	grabSuggest(path, ID, file, client)
}

func processList(maxConc int64, path string, client *redis.Client, worker *sync.WaitGroup) {
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
		go processSingleID(path, scanner.Text(), &wg, file, client)
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
	processList(maxConc, path, client, worker)
}

func argumentParsing(args []string) {
	dbID, _ := strconv.ParseInt(args[4], 10, 64)
	// Connect to Redis server
	client := redis.NewClient(&redis.Options{
		Addr:     args[2],
		Password: args[3],
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
	if len(args) > 5 {
		color.Red("Usage: ./youtube-ig [list of IDs] [CONCURRENCY] [REDIS-HOST] [REDIS-PASSWORD] [REDIS-DB]")
		os.Exit(1)
	} else if len(args) == 5 {
		if _, err := strconv.ParseInt(args[1], 10, 64); err == nil {
			maxConc, _ = strconv.ParseInt(args[1], 10, 64)
		} else {
			color.Red("Usage: ./youtube-ig [list of IDs] [CONCURRENCY] [REDIS-HOST] [REDIS-PASSWORD] [REDIS-DB]")
			os.Exit(1)
		}
	}
	go processList(maxConc, args[0], client, &wg)
	wg.Wait()
}

func main() {
	start := time.Now()
	argumentParsing(os.Args[1:])
	color.Println(color.Cyan("Done in ") + color.Yellow(time.Since(start)) + color.Cyan("!"))
}
