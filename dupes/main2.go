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
	"strings"
	"time"
)

const (
	requestAmt = 10
	firstURL   = "http://realmofthemadgodhrd.appspot.com/account/changePassword?guid="
)

var (
	email       string
	password    string
	tmppassword = "password1234"
	proxies     []string
	randSrc     rand.Source
	once        = false
)

func main() {
	randSrc = rand.NewSource(time.Now().UnixNano())
	fmt.Println("Enter account email...")
	fmt.Scanln(&email)
	fmt.Println("Now enter account password...")
	fmt.Scanln(&password)
	readProxies()
	fmt.Printf("Loaded and using %d proxies\n", len(proxies))
	mainz()
}

func mainz() {
	if once == false {
		once = true
		//sendz()
	}
	fmt.Println("Press enter when you're ready to send requests...")
	fmt.Scanln()
	sendRequests()
}

func sendRequests() {
	//fullURL := rootURL + testString + email + password + "&"
	useOld := false
	for i := 0; i < requestAmt; i++ {
		if useOld == false {
			s, err := getURL(firstURL+email+"&password="+password+"&newPassword="+tmppassword, getRand())
			if err != nil { //if it errors then checkResponse will simply return false and the account wont save
				fmt.Println("Error getting response from deca:", err)
			}
			fmt.Println(s)
			useOld = true
		} else {
			s, err := getURL(firstURL+email+"&password="+tmppassword+"&newPassword="+password, getRand())
			if err != nil { //if it errors then checkResponse will simply return false and the account wont save
				fmt.Println("Error getting response from deca:", err)
			}
			fmt.Println(s)
			useOld = false
		}

	}
	log.Println("Finished!")
	mainz()
}

func getURL(s string, i int) (string, error) {
	req, err := http.NewRequest(http.MethodGet, s, nil)
	if err != nil {
		log.Println("getURL:", err)
	}
	if i != -1 {
		proxyURL, _ := url.Parse(proxies[i])
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
	myClient := &http.Client{}
	ex, err := myClient.Do(req)
	if err != nil {
		return "", err
	}
	body, _ := ioutil.ReadAll(ex.Body)
	ex.Body.Close()
	return string(body), nil
}

func readProxies() {
	f, err := os.Open("list.proxies")
	if err != nil {
		fmt.Println("Error reading proxy file:", err)
		fmt.Scanln()
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		proxies = append(proxies, strings.Split(scanner.Text(), "\n")[0])
	}
}

func getRand() int {
	amtLen := len(proxies)
	for i := 0; i < 1; {
		s := int(randSrc.Int63())
		if s <= amtLen {
			return s
		}
	}
	return -1
}


func getBody(r *http.Response) string {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return ""
	}
	defer r.Body.Close()
	return string(body)
}
