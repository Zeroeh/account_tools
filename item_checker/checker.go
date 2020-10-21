package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

//hastily made script, needs polishing

const (
	rootURL    = "https://realmofthemadgodhrd.appspot.com/"
	listBit    = "char/list?guid=" // guid | password | name
	ipCooldown = 5                 //minutes
	dailyLimit = 59                //max hits before hitting 5 minute cooldown
)

var (
	scanitem    = 0
	proxyIndex  = 0 //the current proxy being used
	ipHits      = 0 //number of hits a proxy has made to appspot
	accountList []string
	proxyList   []string
	banList     []string
	goodList    []string
	emptyList   []string
	userAgents  = []string{
		"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:61.0) Gecko/20100101 Firefox/61.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/64.0.3282.119 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_7_5) AppleWebKit/536.25 (KHTML, like Gecko) Version/6.0 Safari/536.25",
	}
)

func main() {
	rand.Seed(time.Now().UnixNano())
	readProxies()
	readAccounts()
	fmt.Printf("Read %d accounts and %d proxies\n", len(accountList), len(proxyList))
	fmt.Println("Enter the search item ID in decimal...")
	var opt string
	fmt.Scanln(&opt)
	_, err := strconv.Atoi(opt)
	if err != nil {
		fmt.Println("Invalid item ID")
		quit()
	}
	for i := 0; i < len(accountList); i++ {
		email := strings.Split(accountList[i], ":")[0]
		password := strings.Split(accountList[i], ":")[1]
		fullURL := rootURL + "char/list?guid=" + email + "&password=" + password + "&muleDump=true"
		resp, err := getURL(fullURL, proxyIndex)
		if err != nil {
			fmt.Println("Error get:", err)
			proxyIncrement(true)
		}
		proxyIncrement(false)
		if strings.Contains(resp, "maintenance") == true { //bans first
			banList = append(banList, accountList[i])
		} else {
			if strings.Contains(resp, opt) == true { //success 
				goodList = append(goodList, accountList[i])
			} else {
				emptyList = append(emptyList, accountList[i])
			}
		}
		time.Sleep(time.Millisecond * 500)
	}
	fmt.Printf("Good accounts: %d\n", len(goodList))
	fmt.Printf("Empty accounts: %d\n", len(emptyList))
	fmt.Printf("Banned accounts: %d\n", len(banList))
	saveAccounts() //saves empty and banned
}

func saveAccounts() {
	saveBanned()
	saveEmpty()
}

func saveEmpty() {
	file, err := os.OpenFile("empty.search", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		fmt.Printf("Unable to open %s\n", err)
		return
	}
	defer file.Close()
	maxBuf := new(bytes.Buffer)
	for _, v := range emptyList {
		fmt.Fprintf(maxBuf, "%s\n", v)
	}
	_, err = file.Write(maxBuf.Bytes())
	if err != nil {
		fmt.Println("Error writing buffer:", err)
	}
}

func saveBanned() {
	file, err := os.OpenFile("banned.search", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		fmt.Printf("Unable to open %s\n", err)
		return
	}
	defer file.Close()
	maxBuf := new(bytes.Buffer)
	for _, v := range banList {
		fmt.Fprintf(maxBuf, "%s\n", v)
	}
	_, err = file.Write(maxBuf.Bytes())
	if err != nil {
		fmt.Println("Error writing buffer:", err)
	}
}

func readAccounts() {
	// accountList = make(map[int]string)
	f, err := os.Open("login.accounts") //empty.accounts
	if err != nil {
		fmt.Printf("error opening accounts file: %s\n", err)
		quit()
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	index := 0
	for scanner.Scan() {
		//test := strings.Split(scanner.Text(), ":")[0]
		//if test == accounts[index].Email { //if it's already in config then dont save it
		//	break //this is flawed since it doesn't search teh entire accountlist
		//}
		// accountList[index] = scanner.Text()
		accountList = append(accountList, scanner.Text())
		index++
	}
}

func checkProxyBounds() {
	if proxyIndex >= len(proxyList) { //fix proxyindex crash on getURL
		proxyIndex = 0
	}
}

func proxyIncrement(force bool) {
	checkProxyBounds()
	if force == true {
		ipHits = 0
		proxyIndex++
		checkProxyBounds()
	}
	if ipHits >= dailyLimit {
		ipHits = 0
		proxyIndex++
		checkProxyBounds()
	}
}

func readProxies() {
	f, err := os.Open("list.proxies")
	if err != nil {
		log.Printf("error opening proxies file: %s\n", err)
		quit()
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		proxyList = append(proxyList, strings.Split(scanner.Text(), "\n")[0])
	}
}

func getURL(s string, i int) (string, error) {
	defer panicSave()
	//make our request appear "legitimate"
	req, err := http.NewRequest(http.MethodGet, s, nil)
	if err != nil {
		log.Println("getURL:", err)
	}
	req.Header.Set("User-Agent", getRandomHeader())
	if i != -1 {
		var err error
		proxyURL, err := url.Parse(proxyList[i])
		if err != nil {
			return "", err
		}
		myClient := &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			},
		}
		myClient.Timeout = time.Second * 5
		// myClient := &http.Client{}
		// trans := &http.Transport{}
		// trans.Proxy = http.ProxyURL(proxyURL)
		// myClient.Transport = trans
		ex, err := myClient.Do(req)
		if err != nil {
			return "", err
		}
		body, err := ioutil.ReadAll(ex.Body) //don't think we'll error if we got to here
		if err != nil {
			return "", err
		}
		ex.Body.Close()
		return string(body), nil
	}
	//no proxy, use actual ip
	myClient := &http.Client{}
	myClient.Timeout = time.Second * 5
	ex, err := myClient.Do(req)
	if err != nil {
		return "", err
	}
	body, err := ioutil.ReadAll(ex.Body)
	if err != nil {
		return "", err
	}
	ex.Body.Close()
	return string(body), nil
}

func panicSave() {
	if err := recover(); err != nil {
		fmt.Println("Saving from crash")
		saveAccounts()
	}
}

func getRandomHeader() string {
	return userAgents[rand.Int31n(int32(len(userAgents)))]
}

func quit() {
	os.Exit(1)
}
