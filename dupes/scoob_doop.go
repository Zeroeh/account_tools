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

const (
	requestAmt = 600
	firstURL   = "http://realmofthemadgodhrd.appspot.com/account/changePassword?guid="
)

var (
	email       string
	password    string
	tmppassword = "testing1234"
	proxies     []string
	randSrc     rand.Source
	once        = false
	homeIP      = true //proxies are broken I guess
)

func main() {
	randSrc = rand.NewSource(time.Now().UnixNano())
	rand.Seed(time.Now().UnixNano())
	fmt.Println("Enter account email...")
	fmt.Scanln(&email)
	fmt.Println("Now enter account password...")
	fmt.Scanln(&password)
	if homeIP == false {
		readProxies()
		fmt.Printf("Loaded and using %d proxies\n", len(proxies))
	}
	mainz()
}

func mainz() {
	if once == false {
		once = true
	}
	fmt.Println("Press enter when you're ready to send requests...")
	fmt.Scanln()
	sendRequests()
}

func getEmail() string {
	return scrambleEmail()
}

func scrambleEmail() string {
	slice1 := strings.Split(email, "@")[0]
	slice2 := strings.Split(strings.Split(email, "@")[1], ".")[0]
	slice3 := strings.Split(email, ".")[1]
	new1 := scrambleString(slice1)
	new2 := scrambleString(slice2)
	new3 := scrambleString(slice3)
	return new1 + "@" + new2 + "." + new3
}

func scrambleString(s string) string {
	strlen := len(s)
	byteList := make([]byte, 0)
	var newnew byte
	for i := 0; i < strlen; i++ {
		so := string(s[i])
		_, err := strconv.Atoi(so)
		if err != nil {
			newnew = byte(so[0])
			goto next
		}
		switch decideBool() {
		case true:
			newnew = byte(strings.ToUpper(so)[0])
		case false:
			newnew = byte(strings.ToLower(so)[0])
		default:
		}
		next:
		byteList = append(byteList, newnew)
	}
	return string(byteList)
}

func decideBool() bool {
	intop := rand.Int31n(2)
	if intop == 0 {
		return false
	}
	return true
}

func sendRequests() {
	useOld := false
	for i := 0; i < requestAmt; i++ {
		if homeIP == false {
			if useOld == false {
				s, err := getURL(firstURL+email+"&password="+password+"&newPassword="+tmppassword, getRand())
				if err != nil {
					fmt.Println("Error getting response from deca:", err)
				}
				fmt.Println(s)
				useOld = true
			} else {
				s, err := getURL(firstURL+email+"&password="+tmppassword+"&newPassword="+password, getRand())
				if err != nil {
					fmt.Println("Error getting response from deca:", err)
				}
				fmt.Println(s)
				useOld = false
			}
		} else {
			if useOld == false {
				go func() {
					// s, err := getURL(firstURL+getEmail()+"&password="+password+"&newPassword="+tmppassword, -1)
					s, err := getURL(firstURL+getEmail()+"&password="+password+"&do_login=true&isAgeVerified=0", -1)
					if err != nil {
						fmt.Println("Error getting response from deca:", err)
					}
					fmt.Println(s)
					useOld = true
				}()

			} else {
				go func() {
					// s, err := getURL(firstURL+getEmail()+"&password="+tmppassword+"&newPassword="+password, -1)
					s, err := getURL(firstURL+getEmail()+"&password="+password+"&do_login=true&isAgeVerified=1", -1)
					if err != nil {
						fmt.Println("Error getting response from deca:", err)
					}
					fmt.Println(s)
					useOld = false
				}()
			}
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
	ex, err := myClient.Do(req) //zero copy
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
