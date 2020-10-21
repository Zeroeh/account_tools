package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

/*
	acc2bot.go - quick script to turn vault.accounts files and parse them into accounts.json that the bots app can consume
*/

const (
	chosenIP     = "18.218.255.91" //USMidWest2
	botModule    = "receive"
	charID       = 0
	useHTTP      = true
	useSocks     = true
	fetchNewData = false
)

var (
	howMany      = 0
	randSrc      rand.Source
	httpProxies  []string
	socksProxies []string
	accountList  []string
	blackList    []string
	usedList     map[int32]bool
	accounts     []*FileAccount
	botAccounts  []*Account
)

func main() {
	usedList = make(map[int32]bool)
	randSrc = rand.NewSource(time.Now().UnixNano())
	rand.Seed(time.Now().UnixNano())
	fmt.Println("Enter how many accounts to add to the list...")
	var choice string
	fmt.Scanln(&choice)
	i, err := strconv.Atoi(choice)
	if err != nil {
		fmt.Println("Entered wrong thing, bye")
		os.Exit(0)
	}
	howMany = i
	readProxies()
	readBlacklist()
	readAccounts()
	encodeAccounts()
	log.Println("Finished!")
}

func encodeAccounts() {
	accLen := len(accounts)
	trueList := make([]Account, howMany)
	trueListInt := 0
	for k := range usedList { //maps are unordered but this _should_ work as intended
		usedList[k] = false //zero initalize all proxies to not used
	}
	for i := 0; i < accLen; i++ { //search though all the accounts
		//do bounds check for trueList so we dont overflow it
		if trueList[howMany-1].Email != "" { //such a lame way to check... oh well
			goto write
		}
		//check to make sure an account is ready to be parsed
		if accounts[i].Verified == true && accounts[i].Sold == false && accounts[i].Banned == false && accounts[i].Filled == false && trueListInt < howMany {
			account := Account{}
			account.Email = accounts[i].Email
			account.Password = accounts[i].Password
			account.ServerIP = chosenIP
			account.CharID = charID
			account.Module = botModule
			account.UseHTTP = useHTTP
			account.UseSocks = useSocks
			var proxyID int32
			for isGood := false; isGood != true; {
				proxyID = rand.Int31n(int32(len(socksProxies))) //make sure both http and socks proxies use the same ip so we dont mix and match which could lead to high ban rates or other headache-inducing shit
				isGood = checkIfUsed(proxyID)
			}
			usedList[proxyID] = true //since bert fucked up multiboxing we need multiple ips for speedy reconnect
			//note that we dont use the "LastIP" string since it could have been removed or our sub could have ended and we no longer have access to that specific proxy
			account.SockProxy = socksProxies[proxyID]
			account.HTTPProxy = httpProxies[proxyID]
			account.FetchNewData = fetchNewData
			trueList[trueListInt] = account
			trueListInt++
		}
	}
write:
	file, err := os.OpenFile("accounts.json", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		fmt.Printf("Unable to open %s\n", err)
	}
	enc := json.NewEncoder(file)
	enc.SetIndent("", "    ")
	err = enc.Encode(&trueList)
	if err != nil {
		fmt.Printf("error encoding config file: %s\n", err)
		os.Exit(1)
	}
	file.Close()
}

func checkIfUsed(i int32) bool {
	if i < 100 { //dont use an ip that our dupe bots might be using. This cuts the ip list down by a 10th but its worth it for not having any hassles and it essentially negates any bans on the dupe bots from appearing on the receive bots and vice versa
		return false
	}
	if usedList[i] == true {
		return false
	}
	ok := checkBlacklisted(i)
	if ok == false {
		return false
	}
	return true //we checked em all, its good
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

func checkBlacklisted(i int32) bool {
	selected := strings.Split(socksProxies[i], ":")[0]
	for _, x := range blackList {
		if x == selected {
			fmt.Println("Caught blacklisted ip")
			return false
		}
	}
	return true
}

func readBlacklist() {
	f, err := os.Open("blacklist.txt")
	if err != nil {
		fmt.Println("Error opening blacklist:", err)
		os.Exit(1)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		blackList = append(blackList, strings.Split(scanner.Text(), "\n")[0])
	}
}

func readProxies() {
	f, err := os.Open("list.proxies")
	if err != nil {
		fmt.Printf("error opening proxies file: %s\n", err)
		os.Exit(1)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		httpProxies = append(httpProxies, strings.Split(scanner.Text(), "\n")[0])
	}

	f2, err := os.Open("socks.proxies")
	if err != nil {
		fmt.Printf("error opening socks file: %s\n", err)
		os.Exit(1)
	}
	defer f2.Close()
	scanner2 := bufio.NewScanner(f2)
	for scanner2.Scan() {
		socksProxies = append(socksProxies, strings.Split(scanner2.Text(), "socks5://")[1])
	}
}

//Account is imported from json file for bot format
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
