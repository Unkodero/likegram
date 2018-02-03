package main

import (
	"flag"
	"github.com/buger/jsonparser"
	"github.com/fatih/color"
	"golang.org/x/net/proxy"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

var error *color.Color
var info *color.Color
var success *color.Color
var warning *color.Color

var proxyKey string
var proxyDelay int
var accountLogin string
var accountDelay int
var threads int
var threadDelay int

var photoID string
var proxies []string

func main() {
	parseFlags() // Parse options

	// For colored promt
	error = color.New(color.FgRed, color.Bold)
	info = color.New(color.FgBlue, color.Bold)
	success = color.New(color.FgGreen, color.Bold)
	warning = color.New(color.FgYellow, color.Bold)

	// Check api key
	if len(proxyKey) != 32 {
		error.Println("Invalid proxy API key")
		os.Exit(0)
	}

	// Check Instagram login (name?)
	if len(accountLogin) < 3 {
		error.Println("Invalid Instagram login")
		os.Exit(0)
	}

	// Get proxy and start goroutine
	info.Println("Starting proxy update thread")
	go updateProxies()

	// Get last media id and start goroutine
	info.Println("Starting Instagram update thread")
	go getLastPhotoID()

	time.Sleep(time.Second * 5) // Wait for goroutines done their job

	info.Println("Starting likes threads") // THREADS ))))))))

	// Endless goroutines
	var wg sync.WaitGroup
	wg.Add(threads)

	// Starting THREADS)))))
	for threadID := 0; threadID < threads; threadID++ {
		go LikeThread(threadID, &wg)
		time.Sleep(time.Second * 3)
	}

	success.Println("Started", threads, "thread(s)")
	wg.Wait() // Hachiko
}

/**
Parse options
*/
func parseFlags() {
	flag.StringVar(&proxyKey, "proxy", "", "good-proxies.ru API key")
	flag.IntVar(&proxyDelay, "proxy_delay", 500, "Proxy update delay")
	flag.StringVar(&accountLogin, "login", "", "Instagram account login")
	flag.IntVar(&accountDelay, "delay", 600, "Instagram account update in seconds")
	flag.IntVar(&threads, "threads", 2, "Threads count")
	flag.IntVar(&threadDelay, "thread_delay", 15, "Thread delay")
	flag.Parse()
}

/**
Check account for valid and private. And get id of the last media
*/
func getLastPhotoID() {
	for true {
		res, err := http.Get("https://www.instagram.com/" + accountLogin)
		if err != nil {
			panic(err)
		}
		profileBodyBytes, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			panic(err)
		}

		// Parse JSON data from <script> block
		re := regexp.MustCompile("window._sharedData = ({.+});")
		profile := re.FindSubmatch(profileBodyBytes)[1]

		isPrivate, err := jsonparser.GetBoolean(profile, "entry_data", "ProfilePage", "[0]", "user", "is_private")
		// If has no "is_private" field - account not found
		if err != nil {
			error.Println("Invalid login")
			os.Exit(-1)
		}

		if isPrivate {
			error.Println("Account is private")
			os.Exit(-1)
		}

		mediaID, err := jsonparser.GetString(profile, "entry_data", "ProfilePage", "[0]", "user", "media", "nodes", "[0]", "id")
		// If no media
		if err != nil {
			error.Println("Nothing to like")
			os.Exit(-1)
		}

		// Also we need account id for photo id
		profileID, _ := jsonparser.GetString(profile, "entry_data", "ProfilePage", "[0]", "user", "id")

		lastPhotoID := mediaID + "_" + profileID

		// How much likes we have
		likes, _ := jsonparser.GetInt(profile, "entry_data", "ProfilePage", "[0]", "user", "media", "nodes", "[0]", "likes", "count")

		// First interation
		if len(photoID) == 0 {
			success.Println("Set media to like")
			photoID = lastPhotoID
		}

		if len(photoID) == 0 || photoID != lastPhotoID {
			success.Println("Got a new media")
			photoID = lastPhotoID
		}

		success.Println("Now photo have", likes, "likes")
		// Some sleep
		time.Sleep(time.Second * time.Duration(accountDelay))
	}
}

/**
Update proxy
*/
func updateProxies() {
	for true {
		res, err := http.Get("http://api.good-proxies.ru/get.php?type[socks5]=on&count=0&ping=15000&key=" + proxyKey)
		if err != nil {
			panic(err)
		}
		proxyBodyBytes, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			panic(err)
		}

		match, _ := regexp.Match("Вы ввели неверный ключ.", proxyBodyBytes)
		if match {
			error.Println("Invalid proxy key")
			os.Exit(-1)
		}

		// Format proxy list (plain text) to array
		proxies = strings.Split(string(proxyBodyBytes), "\n")
		info.Println("Loaded ", len(proxies), " proxies")

		// Some sleep
		time.Sleep(time.Second * time.Duration(proxyDelay))
	}
}

/**
Return`s maximum random proxy from a list of ~5k proxies
*/
func getRandomProxy() (proxy string) {
	proxy = proxies[rand.Intn(len(proxies))]
	return
}

/**
Main thread (makes request to secret server, with add you account to popularity list)
*/
func LikeThread(id int, wg *sync.WaitGroup) {
	for true {
		// Connect to proxy
		dialer, err := proxy.SOCKS5("tcp", getRandomProxy(), nil, proxy.Direct)
		if err != nil {
			// Invalid proxy
			warning.Println("Thread", id, "have a problem with proxy")
		} else {
			// Proxy good, set transport to http client
			httpTransport := &http.Transport{}
			httpClient := &http.Client{Transport: httpTransport}
			httpTransport.Dial = dialer.Dial

			req, err := http.NewRequest("GET", "http://194.58.115.48/add?lat=45.04280&lon=41.97340&id="+photoID, nil)
			req.Header.Add("User-Agent", "mozilla")
			req.Header.Add("Accept-Language", "en-US,en;q=0.5")
			req.Header.Add("Host", "194.58.115.48")
			req.Header.Add("Connection", "Keep-Alive")
			req.Header.Add("Accept-Encoding", "gzip")
			if err != nil {
				warning.Println("Thread", id, "can`t create request")
			}

			resp, err := httpClient.Do(req)
			if err != nil {
				warning.Println("Thread", id, "can`t create request with current proxy in this interation")
			} else {
				resp.Body.Close()
			}
		}

		time.Sleep(time.Second * time.Duration(threadDelay))
	}
}
