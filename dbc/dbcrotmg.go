package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	godbc "github.com/Zeroeh/go-dbc"
)

/* dbcrotmg.go - verifies accounts with the deathbycaptcha service and sleeps when costs go up
	The script should be able to verify about 600 accounts within a 12 hour period.
known issues:
	app wont sleep if retrying a captcha in the sleep period
*/

type indexJSON struct {
	Index int `json:"index"`
}

const (
	indexFile = "index.json"
)

var (
	listURL []string
	index   int
	success int
	totalSuccess int
	idex    indexJSON
)

func main() {
	fmt.Println("Started")

	c := godbc.CaptchaClient{
		Username: "{redacted}",
		Password: "{redacted}",
		SiteKey:  "6LfYpC0UAAAAABI7pEgdrC8R0tX7goxU_wwSo8Ia",
		PollRate: 10,
	}
	readItems()
	if len(listURL) == 0 {
		fmt.Println("List was empty")
		return
	}
	fmt.Println("Read accounts file...")
	accountIndex := readIndex()
	fmt.Println("Starting from index of", accountIndex)
	for i := accountIndex; accountIndex < len(listURL); i++ {
		//machine local time should not effect this script
		t := time.Now().UTC() //sleep at 1pm utc which is 8pm in thailand and 8am cst
		if t.Hour() == 13 {
			log.Println("Sleeping for 12 hours")
			printSuccess() //only this one will matter as this will hit when we actually run accounts
			time.Sleep(time.Hour * 12)
		} else if t.Hour() == 14 {
			log.Println("Sleeping for 11 hours")
			time.Sleep(time.Hour * 11)
		} else if t.Hour() == 15 {
			log.Println("Sleeping for 10 hours")
			time.Sleep(time.Hour * 10)
		} else if t.Hour() == 16 {
			log.Println("Sleeping for 9 hours")
			time.Sleep(time.Hour * 9)
		} else if t.Hour() == 17 {
			log.Println("Sleeping for 8 hours")
			time.Sleep(time.Hour * 8)
		} else if t.Hour() == 18 {
			log.Println("Sleeping for 7 hours")
			time.Sleep(time.Hour * 7)
		} else if t.Hour() == 19 {
			log.Println("Sleeping for 6 hours")
			time.Sleep(time.Hour * 6)
		} else if t.Hour() == 20 {
			log.Println("Sleeping for 5 hours")
			time.Sleep(time.Hour * 5)
		} else if t.Hour() == 21 {
			log.Println("Sleeping for 4 hours")
			time.Sleep(time.Hour * 4)
		} else if t.Hour() == 22 {
			log.Println("Sleeping for 3 hours")
			time.Sleep(time.Hour * 3)
		} else if t.Hour() == 23 {
			log.Println("Sleeping for 2 hours")
			time.Sleep(time.Hour * 2)
		} else if t.Hour() == 24 {
			log.Println("Sleeping for 1 hour")
			time.Sleep(time.Hour * 1)
		}
		index = i + 1
		accountIndex = i
		c.SiteURL = listURL[i]
		accountID := c.SiteURL[71:] //70 for http 71 for https
		// fmt.Println("AccountID:", accountID)
		attempts := 0
	redo:
		if attempts >= 6 {
			quit()
		}
		id, err := c.Decode(180)
		if err != nil {
			log.Println(err)
			time.Sleep(time.Second * 5)
			attempts++
			goto redo
		}
		if id == 0 {
			attempts++
			time.Sleep(time.Second * 5)
		}
		fmt.Printf("Account ID: %s | Captcha ID: %d\n", accountID, id)
		pollEvent := time.Tick(time.Duration(c.PollRate) * time.Second)
		finishEvent := time.After(time.Duration(c.Timeout) * time.Second)
		for {
			select {
			case <-pollEvent:
				err = c.PollCaptcha(id)
				if err != nil {
					log.Println("Error after polling captcha:", err)
					//hmm...
				}
			case <-finishEvent:
				fmt.Println("Did not solve the captcha in time.")
				c.LastStatus.Text = "nil"
			}
			if c.LastStatus.Text != "nil" && c.LastStatus.Text != "" && len(c.LastStatus.Text) > 10 {
				ok := sendVerify(c.LastStatus.Text, accountID)
				if !ok {
					goto redo
				}
				success++
				saveIndex(index)
				break
			} else if c.LastStatus.Text == "nil" {
				c.LastStatus.Text = ""
				goto redo
			}
		}
		time.Sleep(2500 * time.Millisecond) //give slaves a bit of rest
	}
	fmt.Println("Finished")
}

func sendVerify(t string, id string) bool {
	baseURL := "http://realmofthemadgodhrd.appspot.com/account/v?a="
	action := "&action="
	swear := "I+swear+to+Oryx+I+am+no+bot"
	gcaptcha := "&g-recaptcha-response=" + t
	rotmgURL := baseURL + id + action + url.QueryEscape(swear) + gcaptcha //note: query escaping is optional, but is probably more optimal and less prone to errors
	fmt.Println(rotmgURL)
	resp, err := http.Get(rotmgURL)
	if err != nil {
		log.Println(err)
		return false
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return false
	}
	sbody := string(body)
	if strings.Contains(sbody, "Thank you") == true {
		log.Printf("Account %d completed\n", index)
		return true
	} else if strings.Contains(sbody, "Server Error") {
		log.Printf("Server error on %s\n", rotmgURL)
		return true //our checker will flag the account regardless
	} else {
		log.Printf("Captcha failed on account %s\n", id)
		fmt.Println("Body:", sbody)
		return false
	}
}

func printSuccess() {
	totalSuccess += success
	fmt.Printf("Successfully verified %d accounts since last sleep | %d accounts since startup\n", success, totalSuccess)
	success = 0 //reset
}

func readItems() {
	f, err := os.Open("verify.urls")
	if err != nil {
		fmt.Printf("error opening file: %s\n", err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		listURL = append(listURL, strings.Split(scanner.Text(), "\n")[0])
	}
}

func saveIndex(i int) {
	idex.Index = i
	f, err := os.OpenFile(indexFile, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		fmt.Printf("error opening file: %s\n", err)
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(&idex); err != nil {
		fmt.Printf("error encoding config file: %s\n", err)
		os.Exit(1)
	}
}

func readIndex() int {
	f, err := os.OpenFile(indexFile, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		fmt.Printf("error opening file: %s\n", err)
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&idex); err != nil {
		fmt.Printf("error decoding config file: %s\n", err)
		os.Exit(1)
	}
	return idex.Index
}

func testWrite() {

}

func quit() {
	log.Printf("Shutting down. Accounts verified since startup: %d. Current index is %d\n", totalSuccess, index)
	os.Exit(0)
}

