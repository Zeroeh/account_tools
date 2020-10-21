package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Loginz struct {
	XMLName xml.Name `xml:"NonConsecutive"`
	Text    string   `xml:",chardata"`
	Days    string   `xml:"days,attr"`
	Login   []Logi   `xml:"Login"`
}

type Logi struct {
	Text   string `xml:",chardata"`
	Days   string `xml:"Days"`
	ItemId struct {
		Text     string `xml:",chardata"`
		Quantity string `xml:"quantity,attr"`
	} `xml:"ItemId"`
	Gold string `xml:"Gold"`
	Key  string `xml:"key"`
}

var (
	days *Loginz
)

const (
	setURL = "https://realmofthemadgodhrd.appspot.com/dailyLogin/fetchCalendar?guid=NQPWXHHQFCDRDIFJMMXDCDHJZM@ggnetwork.xyz&password=password1231"
)

func main() {
	resp, err := getURL(setURL)
	if err != nil {
		fmt.Println("Error getting URL:", err, resp)
	}
	idx1 := strings.Index(resp, "<NonConsecutive")
	if idx1 == -1 {
		fmt.Println("First login tag not found")
	}
	idx2 := strings.Index(resp, "</NonConsecutive>")
	if idx2 == -1 {
		fmt.Println("Second tag not found")
	}
	idx2 += len("</NonConsecutive>")
	//fmt.Printf("Target indices: %d | %d\n", idx1, idx2)
	resp2 := resp[idx1:idx2]
	// err = xml.Unmarshal([]byte(resp2), &days)
	rdr := bytes.NewBufferString(resp2)
	err = xml.NewDecoder(rdr).Decode(&days)
	if err != nil {
		fmt.Println("Error marshalling:", err)
	}
	log := fetchDay(2) //2
	fmt.Println(log)
}

func fetchDay(x int) string {
	for i := range days.Login {
		if days.Login[i].Days == strconv.Itoa(x) {
			if days.Login[i].Key != "" {
				return days.Login[i].Key
			}
			return ""
		}
	}
	return ""
}

func getURL(s string) (string, error) {
	// defer panicSave()
	//make our request appear "legitimate"
	req, err := http.NewRequest(http.MethodGet, s, nil)
	if err != nil {
		log.Println("getURL:", err)
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
