package main

/*
	unlocker_checker.go - Checks for how many vaults have been unlocked. Could easily be repurposed to work with char unlockers.
*/

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var (
	accounts   []*Account
	userAgents = []string{
		"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:61.0) Gecko/20100101 Firefox/61.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/64.0.3282.119 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_7_5) AppleWebKit/536.25 (KHTML, like Gecko) Version/6.0 Safari/536.25",
	}
	baseURL = "https://realmofthemadgodhrd.appspot.com/account/verify?guid="
	goodMap = make(map[int]bool)
)

func main() {
	log.Println("Starting...")
	readAccounts()
	//no need to read proxies as we can just grab them from the accounts file. Using the same ip is less suspicious
	for i := range accounts {
		results, err := accounts[i].getURL(baseURL + accounts[i].Email + "&password=" + accounts[i].Password)
		if err != nil {
			fmt.Println("Err getting results from getURL:", err)
		}
		//grab the vault count (all bot vaults should only have 1 potential vault full, rest should be blanks)
		chests := strings.Split(results, "<Chest></Chest>")
		if len(chests) == 140 || len(chests) == 139 { //this account is good to go
			goodMap[i] = true
			if accounts[i].Module != "complete" {
				fmt.Printf("Corrected %d!\n", i)
			}
		} else { //anything else needs the module reset
			fmt.Printf("Account %d has %d vaults\n", i, len(chests))
			goodMap[i] = false
		}
	}
	for i := range goodMap {
		if goodMap[i] == true {
			accounts[i].Module = "complete" //we can be 100% certain that the vault is unlocked
		} else {
			accounts[i].Module = "vaultbegin" //assume that all accounts need to start over from scratch. todo: create a bot module for fixing fucked inventories?
		}
	}
	saveAccounts()
	log.Println("All done!")
}

func saveAccounts() {
	file, err := os.OpenFile("accounts.json", os.O_RDWR|os.O_CREATE, 0666)
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

func readAccounts() {
	f, err := os.Open("accounts.json")
	if err != nil {
		fmt.Printf("error opening accounts file: %s\n", err)
		os.Exit(1)
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&accounts); err != nil {
		fmt.Printf("error decoding config file: %s\n", err)
		os.Exit(1)
	}
}

type Account struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	ServerIP     string `json:"server_ip"`
	FetchNewData bool   `json:"fetch_new_data"`
	CharID       int    `json:"char_id"`
	Module       string `json:"module"`
	UseSocks     bool   `json:"use_socks"`
	SockProxy    string `json:"socks_proxy"`
	UseHTTP      bool   `json:"use_http"`
	HTTPProxy    string `json:"http_proxy"`
}

func (a *Account) getURL(rx string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, rx, nil)
	if err != nil {
		log.Println("getURL:", err)
	}
	req.Header.Set("User-Agent", userAgents[1])
	if a.UseHTTP == true { //should all be true
		var err error
		proxyURL, err := url.Parse(a.HTTPProxy)
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
	} else {
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
}
