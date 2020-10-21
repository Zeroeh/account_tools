package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

//FileAccount | use rfc1123 when formatting times
type FileAccount struct {
	Email           string `json:"email"`
	Password        string `json:"password"`
	Verified        bool   `json:"verified"`
	GameName        string `json:"ign"`
	Creation        string `json:"creation"`
	Filled          bool   `json:"filled"`
	FilledDate      string `json:"filldate"`
	ItemType        string `json:"itemtype"`
	LastStatusCheck string `json:"laststatus"`
	Banned          bool   `json:"banned"`
	Sold            bool   `json:"sold"`
	SellDate        string `json:"selldate"`
	Customer        string `json:"customer"`
	LastIP          string `json:"lastip"`
	Comment         string `json:"comment"`
}

var (
	proxyIndex    = 0 //the current proxy being used
	ipHits        = 0 //number of hits a proxy has made to appspot
	randSrc       rand.Source
	firstProxyHit time.Time
	lastProxyHit  time.Time
	userAgents    = []string{
		"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:61.0) Gecko/20100101 Firefox/61.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/64.0.3282.119 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_7_5) AppleWebKit/536.25 (KHTML, like Gecko) Version/6.0 Safari/536.25",
	}
	accountList  []string
	proxyList    []string
	accountIndex = 0
	accounts     []*FileAccount
)

const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits

	userAgent1 = "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:61.0) Gecko/20100101 Firefox/61.0"
	userAgent2 = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/64.0.3282.119 Safari/537.36"
	userAgent3 = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_7_5) AppleWebKit/536.25 (KHTML, like Gecko) Version/6.0 Safari/536.25"

	accountPassword = "password1231" //emails are going to be randomized so the entropy on those alone
	// should be large enough to prevent brute force attacks
	rootURL    = "https://realmofthemadgodhrd.appspot.com/"
	nameBit    = "account/setName?guid=" // guid | password | name
	ipCooldown = 10                      //minutes
	dailyLimit = 29                      //max accounts per day per ip
)

func main() {
	log.Println("Started")
	randSrc = rand.NewSource(time.Now().UnixNano())
	readProxies()
	readAccounts()
	watchDogLoop()
}

//initLoop is the main
func watchDogLoop() {
	for accountIndex < len(accounts) {
		if proxyIndex == 0 {
			firstProxyHit = time.Now()
		}
		if accounts[accountIndex].GameName == "" {
			sendRequest()
		} else {
			accountIndex++ //we assume it's already named
			fmt.Println("Skipping to", accountIndex)
		}
	}
	saveAccounts() //do one final save incase the last few dont get flushed
}

func sendRequest() {
	email := accounts[accountIndex].Email
	password := accounts[accountIndex].Password
	username := strings.ToUpper(getRandomString(10))
	fullURL := rootURL + nameBit + email + "&password=" + password + "&name=" + username
	if ipHits >= 9 {
		ipHits = 0
		proxyIndex++
	}
	if proxyIndex >= len(proxyList) { //fix proxyindex crash on getURL
		proxyIndex = 0
	}
	resp, err := getURL(fullURL, proxyIndex)
	ipHits++
	if err != nil {
		buf := new(bytes.Buffer)
		fmt.Fprintf(buf, "%s || %s : %s\n", "Namer", proxyList[proxyIndex], err.Error())
		file, err := os.OpenFile("errors.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			log.Println(err)
		}
		defer file.Close()
		if _, err := file.Write(buf.Bytes()); err != nil {
			log.Println(err)
		}
		//assume it's the proxy causing the issue
		proxyIndex++
	}
	if strings.Contains(resp, "Success") == true {
		accounts[accountIndex].GameName = username
		accountIndex++
	} else if strings.Contains(resp, "accountNotFound") == true {
		buf := new(bytes.Buffer)
		fmt.Fprintf(buf, "%s || %s | Account not found\n", "Namer", email)
		file, err := os.OpenFile("errors.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			log.Println(err)
		}
		defer file.Close()
		if _, err := file.Write(buf.Bytes()); err != nil {
			log.Println(err)
		}
	} else if strings.Contains(resp, "notEnoughGold") == true { //already named
		if accounts[accountIndex].GameName == "" {
			accounts[accountIndex].GameName = "N/A"
		}
		accountIndex++ //ASSUME that we're already named
	} else if strings.Contains(resp, "nameIsNotAlpha") == true { //numbers in name
		log.Println("name not alpha:", username)
	} else if strings.Contains(resp, "nameAlreadyInUse") == true {
		log.Println("name in use:", username)
	} else if strings.Contains(resp, "Server Error") == true {

	} else if strings.Contains(resp, "wait") == true {
		proxyIndex++
		ipHits = 0 //new proxy so reset hits
	} else { //called if resp is an error
		buf := new(bytes.Buffer)
		fmt.Fprintf(buf, "Unknown error %s : %s : %s\n", username, email, resp)
		file, err := os.OpenFile("errors.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			log.Println(err)
		}
		defer file.Close()
		if _, err := file.Write(buf.Bytes()); err != nil {
			log.Println(err)
		}
	}
	if accountIndex%10 == 0 {
		saveAccounts()
	}
	time.Sleep(500 * time.Millisecond)
}

func saveAccounts() {
	file, err := os.OpenFile("vault.accounts", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		fmt.Printf("Unable to open %s\n", err)
	}
	//encode the json to file. Note that the output is not human readable. "JSON Tools" extension "Ctrl + Alt + M" will format the json and make it readable
	//note that formatting it is optional and the bot app will read the json file just fine
	//if err := json.NewEncoder(file).Encode(&settings.Accounts); err != nil {
	enc := json.NewEncoder(file)
	enc.SetIndent("", "    ") //prevent the formatting from going out the window
	err = enc.Encode(&accounts)
	if err != nil {
		fmt.Printf("error encoding config file: %s\n", err)
		os.Exit(1)
	}
	file.Close()
}

//resetState flushes buffer, resets proxy states, and sleeps for t hours
func resetState(t int) {
	ipHits = 0
	proxyIndex = 0
	log.Printf("Sleeping for %d hours...\n", t)
	time.Sleep(time.Duration(t) * time.Hour)
}

func logError(app string, message string, item ...string) {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "%s || %s: %s\n", app, item, message)
	file, err := os.OpenFile("errors.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Println(err)
	}
	defer file.Close()
	if _, err := file.Write(buf.Bytes()); err != nil {
		log.Println(err)
	}
}

//getURL... takes index even though its global
func getURL(s string, i int) (string, error) {
	//make our request appear "legitimate"
	req, err := http.NewRequest(http.MethodGet, s, nil)
	if err != nil {
		log.Println("getURL:", err)
	}
	req.Header.Set("User-Agent", userAgent2)
	if i != -1 {
		var err error
		proxyURL, err := url.Parse(proxyList[i])
		if err != nil {
			return "", err
		}
		myClient := &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyURL), //why does this crash?
			},
		}
		// myClient := &http.Client{}
		// trans := &http.Transport{}
		// trans.Proxy = http.ProxyURL(proxyURL)
		// myClient.Transport = trans
		ex, err := myClient.Do(req)
		if err != nil {
			return "", err
		}
		body, _ := ioutil.ReadAll(ex.Body) //don't think we'll error if we got to here
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

func readAccounts() {
	f, err := os.Open("vault.accounts")
	if err != nil {
		log.Printf("error opening vault file: %s\n", err)
		os.Exit(1)
	}
	if err := json.NewDecoder(f).Decode(&accounts); err != nil {
		fmt.Printf("error decoding config file: %s\n", err)
		f.Close()
		os.Exit(1)
	}
	f.Close()
}

func readProxies() {
	f, err := os.Open("list.proxies")
	if err != nil {
		log.Printf("error opening proxies file: %s\n", err)
		os.Exit(1)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		proxyList = append(proxyList, strings.Split(scanner.Text(), "\n")[0])
	}
}

func getRandomFromList(l []string) int {
	return int(rand.Int31n(int32(len(l))))
}

func getRandomString(n int) string {
	b := make([]byte, n)
	for i, cache, remain := n-1, randSrc.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = randSrc.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}
	return string(b)
}
