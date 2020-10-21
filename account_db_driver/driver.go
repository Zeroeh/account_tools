package main

//Specifies how accounts are laid out in structural format

//running this script will convert list.accounts into a json formatted vault.accounts
//the script assumes the converted accounts are fresh out of the create script
//use time.Parse to get back to a time.Time object

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
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
	accountList []string
	accounts    []*FileAccount
)

func main() {
	fmt.Println(time.Now().Format(time.RFC1123))
	readSimple()
	encodeAccounts()
	fmt.Println("Done")
}

func encodeAccounts() {
	accLen := len(accountList)
	trueList := make([]FileAccount, accLen)
	for i := 0; i < accLen; i++ {
		//fill these in with current values for the ones that are hard coded.
		account := FileAccount{}
		account.Email = strings.Split(accountList[i], ":")[0]
		account.Password = strings.Split(accountList[i], ":")[1]
		account.Verified = false
		account.GameName = ""
		account.Creation = time.Now().Format(time.RFC1123) //until the creator script can do this, i'll just do it this way for now
		account.Filled = false
		account.FilledDate = ""
		account.ItemType = ""
		account.LastStatusCheck = ""
		account.Banned = false
		account.Sold = false
		account.SellDate = ""
		account.Customer = ""
		account.LastIP = ""
		account.Comment = ""
		trueList[i] = account
	}
	file, err := os.OpenFile("vault.accounts", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		fmt.Printf("Unable to open %s\n", err)
	}
	//encode the json to file. Note that the output is not human readable. "JSON Tools" extension "Ctrl + Alt + M" will format the json and make it readable
	//note that formatting it is optional and the bot app will read the json file just fine
	enc := json.NewEncoder(file)
	enc.SetIndent("", "    ") //prevent the formatting from going out the window
	err = enc.Encode(&trueList)
	if err != nil {
		fmt.Printf("error encoding config file: %s\n", err)
		os.Exit(1)
	}
	file.Close()
}

func readSimple() {
	f, err := os.Open("login.accounts")
	if err != nil {
		log.Printf("error opening accounts file: %s\n", err)
		os.Exit(1)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		accountList = append(accountList, strings.Split(scanner.Text(), "\n")[0])
	}
}
