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
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/labstack/gommon/color"
)

var checkPre = color.Yellow("[") + color.Green("✓") + color.Yellow("]") + color.Yellow("[")

func processWordResult(word string, page int) int {
	res, err := http.Get("https://www.youtube.com/results?search_query=" + word + "&gl=US&hl=en&disable_polymer=true&page=" + string(page))
	if err != nil {
		color.Println(color.Yellow("[") + color.Red("!") + color.Yellow("]") + color.Yellow("[") + color.Cyan(word) + color.Yellow("]") + color.Red(" No result for word: ") + color.Yellow(word))
		runtime.Goexit()
		return 84
	}
	// defer it!
	defer res.Body.Close()
	// check status, exit if != 200
	if res.StatusCode != 200 {
		color.Println(color.Yellow("[") + color.Red("!") + color.Yellow("]") + color.Yellow("[") + color.Cyan(word) + color.Yellow("]") + color.Red(" No more result for word: ") + color.Yellow(word))
		runtime.Goexit()
		return 84
	}
	body, err := ioutil.ReadAll(res.Body)
	// start goquery in the page
	document, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		runtime.Goexit()
		return 84
	}
	// get IDs
	document.Find("a").Each(func(i int, s *goquery.Selection) {
		if name, _ := s.Attr("dir"); name == "ltr" {
			ID, _ := s.Attr("href")
			if strings.Contains(ID, "/watch?v=") == true && len(ID) == 20 {
				color.Println(checkPre + color.Cyan(word) + color.Yellow("] ") + color.Green("ID found: ") + color.Yellow(ID[9:]))

			}
		}
	})
	return 0
}

func processWord(word string, wg *sync.WaitGroup) {
	defer wg.Done()
	var page = 1
	for i := 1; i != 84; page++ {
		time.Sleep(1 * time.Second)
		if processWordResult(word, page) == 84 {
			i = 84
			runtime.Goexit()
		}
	}
}

func readList(path string) {
	// start workers group
	var wg sync.WaitGroup
	var count int
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
		go processWord(scanner.Text(), &wg)
		if count == 32 {
			wg.Wait()
			count = 0
		}
	}
	// log if error
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	wg.Wait()
}

func startGrabbing(args []string) {
	if _, err := os.Stat(args[0]); err == nil {
		readList(args[0])
	} else {
		fmt.Println("You need to provide a list as argument.")
		os.Exit(1)
	}
}

func main() {
	start := time.Now()
	startGrabbing(os.Args[1:])
	color.Println(color.Cyan("Done in ") + color.Yellow(time.Since(start)) + color.Cyan("!"))
}
