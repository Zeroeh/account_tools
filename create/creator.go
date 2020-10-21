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
	"strings"
	"time"
)

/*	~Notes~
	Currently, I do not know if the 24 hour limit is due to the FIRST hit to register (our first pass thru)
	or if it sets the timer on the LAST hit. And is this limit per request or per the whole ip? (does the first use of the proxy 18 hours ago only need to wait 6 hours or 24 from
		our last hit 10 seconds ago....)
*/

var (
	settings        config
	proxyIndex      = 0 //the current proxy being used
	ipHits          = 0 //number of hits a proxy has made to appspot
	accountsWritten = 0
	flushes         = 0
	randSrc         rand.Source
	firstProxyHit   time.Time
	lastProxyHit    time.Time
	domainList      = []string{ //commented domains are currently being used for other projects or decomissioned
		//"@ggnetwork.xyz",
		//"@realmsupply.xyz",
		//"@darkeye.xyz",
		// "@puewjkhf.xyz",
		// "@iouwkjvn.xyz",
		// "@pojoijhgrel.xyz",
		// "@mkwefgq.xyz",
		// "@qwepiuogfn.xyz",
		// "@xcnborwe.xyz",
		// "@iuhjekjhsad.xyz",
		"@decompiler.xyz",
	}
	userAgents = []string{
		"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:61.0) Gecko/20100101 Firefox/61.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/64.0.3282.119 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_7_5) AppleWebKit/536.25 (KHTML, like Gecko) Version/6.0 Safari/536.25",
	}
	proxyList     []string
	dailyLimitMap map[string]int
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
	rootURL     = "https://realmofthemadgodhrd.appspot.com/"
	registerBit = "account/register?guid=" // guid | newPassword | newGUID | ignore
	flushAmount = 10                       //after writing n accounts, flush to the file
	ipCooldown  = 10                       //minutes
	dailyLimit  = 29                       //max accounts per day per ip
	accountsToRegister = 24 //set the accounts to register. 0 = infinite
)

type config struct {
	Handle    *os.File
	ReadWrite *bufio.ReadWriter
}

func main() {
	log.Println("Started")
	dailyLimitMap = make(map[string]int)
	randSrc = rand.NewSource(time.Now().UnixNano())
	readProxies()
	watchDogLoop()
}

func watchDogLoop() {
	var err error
	settings.Handle, err = os.OpenFile("unverified.accounts", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		log.Println("Error opening account dump file:", err)
	}
	settings.ReadWrite = bufio.NewReadWriter(bufio.NewReader(settings.Handle), bufio.NewWriter(settings.Handle))
	for { //this beast runs 24/7/365
		if accountsToRegister != 0 {
			if accountsToRegister == accountsWritten {
				settings.Handle.Close()
				return
			}
		}
		//run down the list of conditional checks
		if dailyLimitMap[proxyList[len(proxyList)-1]] == dailyLimit {
			lastProxyHit = time.Now()
			for item := range dailyLimitMap {
				dailyLimitMap[item] = 0
			}
			log.Println("Hit 24 hour limit. Sleeping until first proxy is able to run...")
			//we sleep only 5 hours since 29k accounts on 1k proxies with daily limit at 29 is 20.1 hours with 2.5second sleep + network delays
			//might even be able to get away with 4 hours
			time.Sleep(300 * time.Minute)
		}
		if proxyIndex == 0 {
			firstProxyHit = time.Now()
		}
		sendRequest()
	}
}

func sendRequest() {
	email := strings.ToUpper(getRandomString(26)) + domainList[getRandomFromList(domainList)]
	fullURL := rootURL + registerBit + getRandomString(26) + "&newGUID=" + email + "&newPassword=" + accountPassword + "&ignore=4" + "&isAgeVerified=1"
	if ipHits >= 9 {
		ipHits = 0
		if proxyIndex == -1 { //using home ip
			time.Sleep(time.Minute * 10)
		} else {
			proxyIndex++
		}
	}
	if proxyIndex >= len(proxyList) { //fix proxyindex crash on getURL
		proxyIndex = 0
	}
	resp, err := getURL(fullURL, proxyIndex)
	ipHits++
	dailyLimitMap[proxyList[proxyIndex]]++
	if err != nil {
		buf := new(bytes.Buffer)
		fmt.Fprintf(buf, "%s || %s : %s\n", "Creator", proxyList[proxyIndex], err.Error())
		file, err := os.OpenFile("errors.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
		defer file.Close()
		if err != nil {
			log.Println(err)
		}
		if _, err := file.Write(buf.Bytes()); err != nil {
			log.Println(err)
		}
		//assume it's the proxy causing the issue
		if proxyIndex != -1 {
			proxyIndex++
		}
	}
	if strings.Contains(resp, "Success") == true {
		flushed := logAccount(email, accountPassword)
		accountsWritten++
		if flushed == true {
			log.Println("Flushed")
		}
	} else if strings.Contains(resp, "emailAlreadyUsed") == true {
		buf := new(bytes.Buffer)
		fmt.Fprintf(buf, "%s || %s | Email already used\n", "Creator", email)
		file, err := os.OpenFile("errors.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
		defer file.Close()
		if err != nil {
			log.Println(err)
		}
		if _, err := file.Write(buf.Bytes()); err != nil {
			log.Println(err)
		}
	} else if strings.Contains(resp, "repeatError") == true {
		buf := new(bytes.Buffer)
		fmt.Fprintf(buf, "%s || %s:%s | Repeat error\n", "Creator", email, accountPassword)
		file, err := os.OpenFile("errors.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
		defer file.Close()
		if err != nil {
			log.Println(err)
		}
		if _, err := file.Write(buf.Bytes()); err != nil {
			log.Println(err)
		}
	} else if strings.Contains(resp, "MysteryBoxRollModal") == true {
		log.Println("Mysterybox error thing:", email)
	} else if strings.Contains(resp, "quota") == true { //The API call mail.Send() is over quota
		resetState(24)
	} else if strings.Contains(resp, "cancelled") == true { //The API call mail.Send() took too long to respond and was cancelled
		time.Sleep(20 * time.Second)
	} else if strings.Contains(resp, "500 Server Error") == true { //google error
		time.Sleep(30 * time.Second)
	} else if strings.Contains(resp, "ApplicationError") == true { //ApplicationError: 1 Internal error
		buf := new(bytes.Buffer)
		fmt.Fprintf(buf, "%s || %s:%s | Application error\n", "Creator", email, resp)
		file, err := os.OpenFile("errors.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			log.Println(err)
		}
		defer file.Close()
		if _, err := file.Write(buf.Bytes()); err != nil {
			log.Println(err)
		}
	} else if strings.Contains(resp, "wait") == true {
		if proxyIndex != -1 {
			proxyIndex++
		} else {
			time.Sleep(time.Minute * 10)
		}
		ipHits = 0 //new proxy so reset hits
	} else { //called if resp is an error
		buf := new(bytes.Buffer)
		fmt.Fprintf(buf, "Unknown error %s : %s\n", email, resp)
		file, err := os.OpenFile("errors.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			log.Println(err)
		}
		defer file.Close()
		if _, err := file.Write(buf.Bytes()); err != nil {
			log.Println(err)
		}
	}
	time.Sleep(2500 * time.Millisecond)
}

//resetState flushes buffer, resets proxy states, and sleeps for t hours
func resetState(t int) {
	settings.ReadWrite.Flush() //flush any buffered accounts
	ipHits = 0
	if proxyIndex != -1 {
		proxyIndex = 0
	}
	for _, val := range proxyList {
		dailyLimitMap[val] = 0 //reset proxy states
	}
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
		return string(body), nil
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

//logAccount returns true if the buffer was flushed on this call
func logAccount(email string, password string) bool {
	buf := new(bytes.Buffer)
	if email != "" && password != "" { //logging an actual account
		fmt.Fprintf(buf, "%s:%s\n", email, password)
	} else if email != "" && password == "" { //logging a singular string
		fmt.Fprintf(buf, "%s\n", email) //log whatever is in "email"
	} else { //email and password were both likely blank
	}
	if _, err := settings.ReadWrite.Write(buf.Bytes()); err != nil {
		log.Println("Error writing account:", err)
	}
	log.Println("Wrote account:", accountsWritten)
	if accountsWritten%flushAmount == 5 {
		err := settings.ReadWrite.Flush()
		if err != nil {
			log.Println("Error on flush:", err)
		}
		return true
	} else {
		return false
	}
}

func readProxies() {
	f, err := os.Open("list.proxies")
	if err != nil {
		if strings.Contains(err.Error(), "no such") {
			fmt.Println("Couldn't find proxy file. Using home ip...")
			return
		}
		log.Printf("error opening proxies file: %s\n", err)
		os.Exit(1)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		proxyList = append(proxyList, strings.Split(scanner.Text(), "\n")[0])
	}
	for _, val := range proxyList {
		dailyLimitMap[val] = 0
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
