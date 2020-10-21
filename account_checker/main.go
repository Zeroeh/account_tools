package main

/*
	account_checker: windows app for resellers to see how many items they have available without having to load them into muledump
*/

//build with: GOOS=windows go build -ldflags="-s -w"

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	periodWaitMS           = 2000
	fiveMinuteSleepSeconds = 301
	cooldownCount          = 59
	rootURL                = "https://realmofthemadgodhrd.appspot.com/char/list?guid="
	passURL                = "&password="
	saveFileMsg            = "Enter the name of the file to save the accounts to. Example: new.accounts"
	readFileMsg            = "Enter the name of the file to scan. Example: items.accounts\n The file must be in the same directory as this exe."
	saveFileDlg            = "Would you like to save the new file? Y/n"
	quitDlg                = "Exiting in a few seconds..."
	quitSleepTime          = 10
)

var (
	oldAccounts map[string]string
	newAccounts map[string]string
)

func main() {
	oldAccounts = make(map[string]string)
	newAccounts = make(map[string]string)
	fmt.Println(readFileMsg)
	var choice string
	fmt.Scanln(&choice)
	readAccounts(choice)

	checkAccounts()

	fmt.Println(saveFileDlg)
	var choice2 string
	fmt.Scanln(&choice2)
	switch choice2 {
	case "Y":
		fmt.Println(saveFileMsg)
		var choice3 string
		fmt.Scanln(&choice3)
		saveNewFile(choice3)
	case "y":
		fmt.Println(saveFileMsg)
		var choice3 string
		fmt.Scanln(&choice3)
		saveNewFile(choice3)
	default:
	}
}

func checkAccounts() {
	fmt.Println("Checking accounts...")
	rateLimit := 0
	for k, v := range oldAccounts {
		if rateLimit >= cooldownCount {
			log.Println("Sleeping for 5 minutes to bypass temp ban...")
			rateLimit = 0
			time.Sleep(time.Second * fiveMinuteSleepSeconds)
		}
		fullURL := rootURL + k + passURL + v
		s, err := getURL(fullURL)
		if err != nil { //if it errors then checkResponse will simply return false and the account wont save
			fmt.Println("Error getting response from deca:", err)
		}
		rateLimit++
		ok := checkResponse(s, k)
		if ok == true {
			newAccounts[k] = v
		}
		time.Sleep(time.Millisecond * periodWaitMS)
	}
}

func checkResponse(r string, e string) bool {
	if strings.Contains(r, "<Chars") == true {
		if strings.Contains(r, "2990,2990,2990,2990,2990,2990,2990") == true { //decades
			fmt.Printf("%s has Decades\n", e)
			return true
		}
		if strings.Contains(r, "8608,8608,8608,8608,8608,8608,8608") == true { //etherites
			fmt.Printf("%s has Etherites\n", e)
			return true
		}
		if strings.Contains(r, "9060,9060,9060,9060,9060,9060,9060") == true { //fairies
			fmt.Printf("%s has Fairy Plates\n", e)
			return true
		}
		if strings.Contains(r, "9063,9063,9063,9063,9063,9063,9063") == true { //pixies
			fmt.Printf("%s has Pixies\n", e)
			return true
		}
	} else {
		fmt.Printf("%s: Got bad response: %s\n", e, r)
		fmt.Println("Skipping this account")
		return false
	}
	fmt.Printf("%s came out empty\n", e)
	return false
}

func getURL(s string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, s, nil)
	if err != nil {
		log.Println("getURL:", err)
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

func readAccounts(fname string) {
	f, err := os.Open(fname)
	if err != nil {
		fmt.Println("Error opening file:", err)
		quit()
	}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		arrs := strings.Split(scanner.Text(), ":")
		if len(arrs) < 2 {
			fmt.Println("Skipping bad line")
		} else {
			email := arrs[0]
			password := arrs[1]
			if email == "" || password == "" {
				fmt.Println("Skipping empty line")
			} else {
				oldAccounts[email] = password
			}
		}

	}
	f.Close()
	fmt.Printf("Read %d accounts\n", len(oldAccounts))
}

func saveNewFile(fname string) {
	f, err := os.OpenFile(fname, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		fmt.Println("Error creating new file:", err)
		quit()
	}
	for i, v := range oldAccounts {
		email := i
		password := v
		fmt.Fprintf(f, "%s:%s\n", email, password)
	}
	f.Close()
}

func quit() {
	fmt.Println(quitDlg)
	time.Sleep(time.Second * quitSleepTime)
	os.Exit(0)
}
