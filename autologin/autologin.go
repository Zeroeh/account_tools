package main

import (
	"bufio"
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

/*
	autologin.go - used to increment daily login calendar for thousands of accounts without actually logging into them
	todo: add file logging for errors
	todo: add accounts per hour. use a history map to show acccounts per hour by the hour
		86000 -> seconds per day
		43000 -> above divided by 2, accounts log twice daily
		21500 -> above / 2, accounts for delays
		~24440 seconds to run 12220 accounts
		Total: $611000 for 12220 accounts
		Total: $877600 for 17552 accounts
		17500 accounts is ALMOST too much for a 12 hour period, in real world testing (a little over 30 minutes remained)
*/

const (
	rateLimit     = 59
	secondsPerDay = 86000 //true 24 hours is 86400
)

var (
	msSleep    = 700 //time in ms to sleep after each account
	randSrc    rand.Source
	userAgents = []string{
		"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:61.0) Gecko/20100101 Firefox/61.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/64.0.3282.119 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_7_5) AppleWebKit/536.25 (KHTML, like Gecko) Version/6.0 Safari/536.25",
	}
	baseURL      = "https://realmofthemadgodhrd.appspot.com/char/list?guid="
	usingProxies = true //can probably leave this false if only checking a handful of accounts
	newDay       = false //hardcode this to true so that accounts are checked on launch
	verbose      = true //set to true if we havent used the script in some time, to make sure everything works
	runTimes     = 0
	counter      = 0
	accountLen   = 0
	proxyLen     = 0
	ipHits       = 0
	proxyIndex   = 0
	checkRounds  = 1
	sleepHours   = 6 //todo: make this change based on the # of accounts being checked
	//usually the speed at which 1 account completes thru proxy is ~1 second (real-world is ~1.5 seconds)
	timeStart time.Time
	timeStop  time.Time
	lastRound time.Time
	accounts  []string
	proxyList []string
)

func main() {
	randSrc = rand.NewSource(time.Now().UnixNano())
	rand.Seed(time.Now().UnixNano())
	if usingProxies == true {
		readProxies()
		proxyLen = len(proxyList)
		fmt.Printf("Loaded %d proxies\n", proxyLen)
	}
	readAccounts()
	accountLen = len(accounts)
	fmt.Printf("Loaded %d accounts\n", accountLen)
	go changeSleep()
	go launchThread()
	for { //loop endlessly
	onceagain:
		if newDay == true { //I should really make this its own function, but this works
			if counter >= accountLen {
				runTimes++
				lastRound = time.Now()
				if runTimes == checkRounds { //check each account twice daily
					runTimes = 0
					ipHits = 0
					counter = 0
					newDay = false
					verbose = false
					timeStop = time.Now()
					elapsed := time.Since(timeStart).Hours()
					fmt.Printf("Elapsed %f hours since last start\n", elapsed)
					fmt.Printf("Started @ %s | Ended @ %s\n", timeStart.String(), timeStop.String())
					log.Printf("Waiting for the next day to start... (in about %d hours)\n", 24-time.Now().UTC().Hour())
					goto onceagain
				}
				log.Println("Beginning next round...")
				ipHits = 0
				counter = 0
			}
			if ipHits >= rateLimit && usingProxies == false { //for home ip
				ipHits = 0
				log.Println("Sleeping for 6 minutes")
				time.Sleep(time.Minute * 6)
			}
			email, password := getAccount()
			fullURL := baseURL + email + "&password=" + password + "&muleDump=true&do_login=true"
		tryagain:
			var resp string
			var err error
			if usingProxies == true {
				resp, err = getURL(fullURL, proxyIndex)
			} else {
				resp, err = getURL(fullURL, -1)
			}
			ipHits++
			if err != nil {
				if strings.Contains(err.Error(), "Client.Timeout") == true {
					log.Printf("Proxy %d timed out (%s)\n", proxyIndex, proxyList[proxyIndex])
					// log.Printf("Proxy %d timed out\n", proxyIndex)
				} else {
					log.Println("Error getting response:", err)
				}
				proxyIncrement(true)
				goto tryagain
				//just skip over the account and hope it's good next time
			}
			if usingProxies == true {
				proxyIncrement(false)
			}
			if strings.Contains(resp, "Chars") == true {
				if verbose == true {
					log.Printf("Success! %s\n", email)
				}
			} else {
				if strings.Contains(resp, "error") == true {
					proxyIncrement(true)
					goto tryagain
				} else {
					log.Printf("Failure? %s | %s\n", email, resp)
				}
			}
			counter++                                             //next account, increments even if char list returns weird shit
			time.Sleep(time.Millisecond * time.Duration(msSleep)) //hopefully this isn't too fast
		} else { //keep checking to make sure we begin once the new day is in effect
			time.Sleep(time.Second * 1)
		}
	}
}

func getAccount() (string, string) {
	selected := accounts[counter]
	email := strings.Split(selected, ":")[0]
	password := strings.Split(selected, ":")[1]
	return email, password
}

func changeSleep() {
	for {
		var choice string
		fmt.Scanln(&choice)
		amt, err := strconv.Atoi(choice)
		if err != nil {
			switch choice {
			case "verbose":
				if verbose == true {
					verbose = false
					fmt.Println("Switched to quiet output")
				} else {
					verbose = true
					fmt.Println("Switched to verbose output")
				}
			case "time":
				fmt.Println("Current UTC hour is", time.Now().UTC().Hour())
			case "stats":
				log.Printf("Current stats:\n")
				fmt.Printf(" Checking Accounts: ")
				if newDay == true {
					fmt.Printf("true\n")
				} else {
					fmt.Printf("false\n")
				}
				fmt.Printf(" Account Index: %d of %d\n", counter, accountLen)
				fmt.Printf(" Current Round: %d\n", runTimes+1)
				fmt.Printf(" Current Proxy: %d\n", proxyIndex)
				fmt.Printf(" Sleep Time: %dms\n", msSleep)
				if runTimes+1 == checkRounds { //for multi rounds
					fmt.Printf(" Finished last round @ %s and %s\n", lastRound, lastRound.UTC())
				}
			default:
				fmt.Println("Bad entry")
			}
			goto end
		}
		fmt.Printf("Setting new sleep to %d ms\n", amt)
		msSleep = amt
	end:
	}
}

//this function is to ensure that we run on the new day
func launchThread() {
	for {
		switch time.Now().UTC().Hour() {
		case 0: //new day, this would be 12am for 12-hour clocks
			newDay = true
			log.Println("New day!")
			timeStart = time.Now()
			time.Sleep(time.Hour * 1)
		default:
			time.Sleep(time.Second * 1)
		}
	}
}

func readAccounts() {
	f, err := os.Open("login.accounts")
	if err != nil {
		fmt.Printf("error opening login file: %s\n", err)
		os.Exit(1)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		accounts = append(accounts, strings.Split(scanner.Text(), "\n")[0])
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
	// defer panicSave()
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

func checkProxyBounds() {
	if proxyIndex >= len(proxyList) { //fix proxyindex crash on getURL
		proxyIndex = 0
	}
}

//These are global so no need to include each time in function. force bool forces a proxy change
func proxyIncrement(force bool) {
	checkProxyBounds()
	if force == true {
		ipHits = 0
		proxyIndex++
		checkProxyBounds()
	}
	if ipHits >= rateLimit {
		ipHits = 0
		proxyIndex++
		checkProxyBounds()
	}
}

func getRandomHeader() string {
	return userAgents[rand.Int31n(int32(len(userAgents)))]
}

func quit() {
	os.Exit(0)
}
