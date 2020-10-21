package main

/*
	lister.go
		Grabs char/list for an account and does various things based on the information received
		We'll grab as much information as possible from here so that we dont have to go in game and get it
		Last minute decision to also make this the ordering script so that I can do orders while mules are getting checked

*/

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits

	userAgent1 = "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:61.0) Gecko/20100101 Firefox/61.0"
	userAgent2 = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/64.0.3282.119 Safari/537.36"
	userAgent3 = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_7_5) AppleWebKit/536.25 (KHTML, like Gecko) Version/6.0 Safari/536.25"

	rootURL    = "https://realmofthemadgodhrd.appspot.com/"
	listBit    = "char/list?guid=" // guid | password | name
	ipCooldown = 5                 //minutes
	dailyLimit = 59                //max hits before hitting 5 minute cooldown
)

var (
	proxyIndex    = 0 //the current proxy being used
	ipHits        = 0 //number of hits a proxy has made to appspot
	sCode         = 0
	randSrc       rand.Source
	firstProxyHit time.Time
	lastProxyHit  time.Time
	theCart       = make(map[int]*FileAccount)
	userAgents    = []string{
		"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:61.0) Gecko/20100101 Firefox/61.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/64.0.3282.119 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_7_5) AppleWebKit/536.25 (KHTML, like Gecko) Version/6.0 Safari/536.25",
	}
	banList      []int
	accountList  []string
	proxyList    []string
	accountIndex = 0
	startTime    time.Time
	accounts     []*FileAccount
	botAccounts  []*BotAccount
)

func main() {
	startTime = time.Now()
	randSrc = rand.NewSource(time.Now().UnixNano())
	fmt.Printf("Welcome to the RealmSupply MuleDump/Ordering script...\n")
	fmt.Printf("To start, select an option...\n")
	fmt.Printf("  1. Select a range of accounts to check (DEPRECATED, USE OPTION 4 INSTEAD)\n")
	fmt.Printf("  2. Go through all accounts and check them\n")
	fmt.Printf("  3. Prepare an order\n")
	fmt.Printf("  4. Import an accounts.json file and check all of the accounts in the file\n")
	fmt.Printf("  5. Get the total number of available accounts containing a specified item\n")
	fmt.Printf("  6. Check all currently unverified accounts (NOT WORKING, STILL WIP)\n")
	fmt.Printf("  7. Check all currently verified accounts that are not sold (checks empty + filled accounts)\n")
	fmt.Printf("  8. Check for bans. This only checks the current state of the vault. Run option 2 to update ban statuses.\n")
	fmt.Printf("  9. See how many verified and unverified accounts are available.\n")
	fmt.Printf(" 10. Check all proxy connections to RotMG Appspot\n")
	fmt.Printf("  0. Exit the script\n")
	var chosen string
	fmt.Scanln(&chosen)
	switch chosen {
	case "0":
		quit()
	case "1":
		sCode = 1
	case "2":
		sCode = 2
	case "3":
		sCode = 3
	case "4":
		sCode = 4
	case "5":
		sCode = 5
	case "6":
		sCode = 6
	case "7":
		sCode = 7
	case "8":
		sCode = 8
	case "9":
		sCode = 9
	case "10":
		sCode = 10
	default:
		fmt.Printf("Got unknown result %s, restarting script\n", chosen)
		main()
	}
	runEvent()
	cleanUp()
}

func runEvent() {
	//do some prep work...
	readProxies()
	fmt.Printf("Loaded and using %d proxies\n", len(proxyList))
	readAccounts()
	fmt.Printf("Loaded %d accounts\n", len(accounts))
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go sigHandler(signals)

	switch sCode {
	case 1: //check a range of accounts
		var choice string
		fmt.Println("Enter a range of accounts to check... Example: 0-225 or 111-23333 and ect")
		fmt.Scanln(&choice)
		var item1, item2 string
		item1 = strings.Split(choice, "-")[0]
		item2 = strings.Split(choice, "-")[1]
		var icode1, icode2 int
		var err, err1 error
		icode1, err = strconv.Atoi(item1)
		icode2, err1 = strconv.Atoi(item2)
		if err != nil || err1 != nil {
			fmt.Println("Had issues converting account range... Exitting...")
			quit()
		}
		if icode1 > icode2 {
			fmt.Println("First index is bigger than the second... bakka na!")
			quit()
		}
		if icode2 > len(accounts) {
			fmt.Println("Second index is bigger than account array length...")
			quit()
		}
		fmt.Printf("Selected range is %d through %d\n", icode1, icode2)
		runAccounts(icode1, icode2, false)
	case 2: //check all accounts
		fmt.Println("Do you want to force check all accounts, including sold and recently checked accounts? Y/n")
		runAccounts(-1, -1, false)
	case 3: //prepare an order
		startAnOrder()
	case 4:
		importFile()
	case 5:
		checkItem()
	case 6:
		checkUnverified()
	case 7:
		checkVerified()
	case 8:
		readBans()
	case 9:
		readVerificationStatus()
	case 10:
		checkProxies()
	default:
		fmt.Println("Got bad sCode in runEvent... Exitting...")
		quit()
	}
}

func readVerificationStatus() {
	fmt.Println("Checking all empty accounts for verified status...")
	emptyVerified := 0
	unverified := 0
	for i := range accounts {
		if accounts[i].Verified == true && accounts[i].Filled == false {
			emptyVerified++
		}
		if accounts[i].Verified == false {
			unverified++
		}
	}
	fmt.Printf("Currently have %d empty verified accounts and %d unverified accounts\n", emptyVerified, unverified)
}

func readBans() {
	fmt.Println("Gathering up all banned accounts...")
	total := 0
	for i := range accounts {
		if accounts[i].Banned == true {
			total++
			banList = append(banList, i)
		}
	}
	fmt.Printf("Currently have %d total banned accounts. Would you like to export the indices of the banned accounts? Y/n\n", total)
	var choice string
	fmt.Scanln(&choice)
	switch choice {
	case "Y":
		exportBannedAccounts()
	case "y":
		exportBannedAccounts()
	default:

	}
}

func checkVerified() {

}

func checkUnverified() { //incomplete, should start from the index and check the rest of the accounts
	fmt.Printf("Grabbing index from dbc script... ")
	f, err := os.Open("index.json")
	if err != nil {
		fmt.Println("Error opening file:", err)
	}
	defer f.Close()
	var s struct {
		Index int `json:"Index"`
	}
	err = json.NewDecoder(f).Decode(&s)
	if err != nil {
		fmt.Println("Error decoding json:", err)
	}
	end := s.Index
	fmt.Printf("index is %d\n", end)
}

func checkItem() {
	pixieTotal := 0
	decadeTotal := 0
	etheriteTotal := 0
	plateTotal := 0
	spellTotal := 0
	emptyTotal := 0
	unknownTotal := 0
	for i := range accounts {
		if accounts[i].Sold == false {
			switch accounts[i].ItemType {
			case "decades": decadeTotal++
			case "pixies": pixieTotal++
			case "etherites": etheriteTotal++
			case "fairies": plateTotal++
			case "pierces": spellTotal++
			case "empty": emptyTotal++
			default: unknownTotal++
			}
		}
	}
	fmt.Printf("%d accounts found with %d pixies.\n", pixieTotal, pixieTotal*8)
	fmt.Printf("%d accounts found with %d decades.\n", decadeTotal, decadeTotal*8)
	fmt.Printf("%d accounts found with %d etherites.\n", etheriteTotal, etheriteTotal*8)
	fmt.Printf("%d accounts found with %d plates.\n", plateTotal, plateTotal*8)
	fmt.Printf("%d accounts found with %d spells.\n", spellTotal, spellTotal*8)
	fmt.Printf("%d accounts found with %d empty.\n", emptyTotal, emptyTotal*8)
	fmt.Printf("%d accounts found with %d blank or unknown item.\n", unknownTotal, unknownTotal*8)
}

func importFile() {
	fmt.Println("Enter the file name to import... Note that the file must be a json file in the format that the bots use. Enter nothing to scan 'accounts.json'")
	var choice string
	fmt.Scanln(&choice)
	if choice == "" {
		choice = "accounts.json"
	}
	fmt.Println("Opening", choice)
	f, err := os.Open(choice)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer f.Close()
	err = json.NewDecoder(f).Decode(&botAccounts)
	if err != nil {
		fmt.Println("Error decoding json:", err)
	}
	log.Printf("Going through %d accounts...\n", len(botAccounts))
	for i := range botAccounts {
		ok := false
		storeIndex := 0 //make an index so we can set the data at the account index
		//not sure if this is the _best_ way but it will have to suffice
		//search thru the entire accounts list for the email and if it exists, we can continue
		for x := range accounts {
			if accounts[x].Email == botAccounts[i].Email {
				ok = true
				storeIndex = x
				fmt.Printf("%d @ accounts[%d]: ", i+1, x)
				break //dont need to continue looping
			}
		}
		if ok == true { //it's in the database so we can refresh the data
			fmt.Printf("Checking... ")
			fullURL := rootURL + listBit + botAccounts[i].Email + "&password=" + botAccounts[i].Password
		checkagainz:
			resp, err := getURL(fullURL, proxyIndex)
			ipHits++
			if err != nil {
				buf := new(bytes.Buffer)
				fmt.Fprintf(buf, "%s || %s : %s\n", "Lister", proxyList[proxyIndex], err.Error())
				file, err := os.OpenFile("errors.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
				if err != nil {
					log.Println(err)
				}
				defer file.Close()
				if _, err := file.Write(buf.Bytes()); err != nil {
					log.Println(err)
				}
				//assume it's the proxy causing the issue
				proxyIncrement(true)
				goto checkagainz
			}
			checkAgain := checkResponse(resp, storeIndex)
			if checkAgain == true {
				proxyIncrement(true) //assume the proxy is the issue and change it
				goto checkagainz
			} else {
				proxyIncrement(false) //normal proxy switching behavior
			}
			time.Sleep(1500 * time.Millisecond) //so we dont get 1 minute ban
			fmt.Printf("Ok!\n")
		} else {
			fmt.Printf("Found no email for %d in database matching '%s'\n", i, botAccounts[i].Email)
		}
	}
}

func runAccounts(i1, i2 int, force bool) {
	if i1 == -1 && i2 == -1 { //run through ALL accounts
		log.Println("Running through all accounts...")
		for i := 0; i < len(accounts); i++ {
			continueacct := true
			tx, err := time.Parse(time.RFC1123, accounts[i].LastStatusCheck)
			if err != nil {
				continueacct = true
			}
			_ = tx //do time calcs
			if accounts[i].Sold == true {
				continueacct = false
			}
			if force == true {
				continueacct = true
			}
			if continueacct == true {
				fullURL := rootURL + listBit + accounts[i].Email + "&password=" + accounts[i].Password + "&muleDump=true"
			checkagain:
				resp, err := getURL(fullURL, proxyIndex)
				ipHits++
				if err != nil {
					buf := new(bytes.Buffer)
					fmt.Fprintf(buf, "%s || %s : %s\n", "Lister", proxyList[proxyIndex], err.Error())
					file, err := os.OpenFile("errors.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
					if err != nil {
						log.Println(err)
					}
					if _, err := file.Write(buf.Bytes()); err != nil {
						log.Println(err)
						// file.Close()
					}
					file.Close()
					//assume it's the proxy causing the issue
					proxyIncrement(true)
					goto checkagain
				}
				checkAgain := checkResponse(resp, i)
				if checkAgain == true {
					proxyIncrement(true) //assume the proxy is the issue and change it
					goto checkagain
				} else {
					proxyIncrement(false) //normal proxy switching behavior
				}
				//fmt.Printf("Checked %s\n", accounts[i].Email)
				time.Sleep(1000 * time.Millisecond) //so we dont get 1 minute ban
			}
		}
	} else { //only go through the selected range (THIS IS NOW DEPRECATED AND UNSUPPORTED! USE AT OWN RISK!!)
		log.Printf("Running through %d accounts...\n", i2-i1)
		for ; i1 < i2; i1++ {
			continueacct := true
			tx, err := time.Parse(time.RFC1123, accounts[i1].LastStatusCheck)
			if err != nil {
				continueacct = true
			}
			_ = tx //do time calcs
			if accounts[i1].Sold == true {
				continueacct = false
			}
			if force == true {
				continueacct = true
			}
			if continueacct == true {
				fullURL := rootURL + listBit + accounts[i1].Email + "&password=" + accounts[i1].Password + "&muleDump=true"
			checkagaintwo:
				resp, err := getURL(fullURL, proxyIndex)
				ipHits++
				if err != nil {
					buf := new(bytes.Buffer)
					fmt.Fprintf(buf, "%s || %s : %s\n", "Lister", proxyList[proxyIndex], err.Error())
					file, err := os.OpenFile("errors.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
					if err != nil {
						log.Println(err)
					}
					defer file.Close()
					if _, err := file.Write(buf.Bytes()); err != nil {
						log.Println(err)
						// file.Close()
					}
					file.Close()
					//assume it's the proxy causing the issue
					proxyIncrement(true)
					goto checkagaintwo
				}
				checkAgain := checkResponse(resp, i1)
				if checkAgain == true {
					proxyIncrement(true) //assume the proxy is the issue and change it
					goto checkagaintwo
				} else {
					proxyIncrement(false) //normal proxy switching behavior
				}
				fmt.Printf("%d @ %s\n", i1, accounts[i1].Email)
				time.Sleep(1000 * time.Millisecond) //so we dont get 1 minute ban
			}
		}
	}
}

func startAnOrder() {
	printOrderMenu()
	var chosen string
	fmt.Scanln(&chosen)
	switch chosen {
	case "1":
		fmt.Println("Choosing to add some Decades")
		fmt.Println("Just enter the number of accounts to add to the cart")
	decadelabel:
		var amt string
		fmt.Scanln(&amt)
		real, err := strconv.Atoi(amt)
		if err != nil {
			fmt.Println("Bad input, try again")
			goto decadelabel
		}
		ok := addAccounts("decades", real)
		if ok {
			fmt.Println("Successfully added items to cart. Returning to main menu")
		} else {
			fmt.Println("Not enough accounts, invalid input, or some other shit. Note that accounts could still have been added to cart. Run 'Check Cart' to see")
		}
		startAnOrder()
	case "2":
		fmt.Println("Choosing to add some Pixie Swords")
		fmt.Println("Just enter the number of accounts to add to the cart")
	pixielabel:
		var amt string
		fmt.Scanln(&amt)
		real, err := strconv.Atoi(amt)
		if err != nil {
			fmt.Println("Bad input, try again")
			goto pixielabel
		}
		ok := addAccounts("pixies", real)
		if ok {
			fmt.Println("Successfully added items to cart. Returning to main menu")
		} else {
			fmt.Println("Not enough accounts, invalid input, or some other shit. Note that accounts could still have been added to cart. Run 'Check Cart' to see")
		}
		startAnOrder()
	case "3":
		fmt.Println("Choosing to add some Etherites")
		fmt.Println("Just enter the number of accounts to add to the cart")
	etheritelabel:
		var amt string
		fmt.Scanln(&amt)
		real, err := strconv.Atoi(amt)
		if err != nil {
			fmt.Println("Bad input, try again")
			goto etheritelabel
		}
		ok := addAccounts("etherites", real)
		if ok {
			fmt.Println("Successfully added items to cart. Returning to main menu")
		} else {
			fmt.Println("Not enough accounts, invalid input, or some other shit. Note that accounts could still have been added to cart. Run 'Check Cart' to see")
		}
		startAnOrder()
	case "4":
	case "5":
	case "6":
		fmt.Println("Choosing to add some Fairy Plates")
		fmt.Println("Just enter the number of accounts to add to the cart")
	fairielabel:
		var amt string
		fmt.Scanln(&amt)
		real, err := strconv.Atoi(amt)
		if err != nil {
			fmt.Println("Bad input, try again")
			goto fairielabel
		}
		ok := addAccounts("fairies", real)
		if ok {
			fmt.Println("Successfully added items to cart. Returning to main menu")
		} else {
			fmt.Println("Not enough accounts, invalid input, or some other shit. Note that accounts could still have been added to cart. Run 'Check Cart' to see")
		}
		startAnOrder()
	case "7":
	case "8":
	case "9":
		fmt.Println("Choosing to add some Mules")
		fmt.Println("Just enter the number of accounts to add to the cart")
	emptylabel:
		var amt string
		fmt.Scanln(&amt)
		real, err := strconv.Atoi(amt)
		if err != nil {
			fmt.Println("Bad input, try again")
			goto emptylabel
		}
		ok := addAccounts("empty", real)
		if ok {
			fmt.Println("Successfully added items to cart. Returning to main menu")
		} else {
			fmt.Println("Not enough accounts, invalid input, or some other shit. Note that accounts could still have been added to cart. Run 'Check Cart' to see")
		}
		startAnOrder()
	case "96":
		fmt.Printf("Currently have %d accounts in the cart.\n", len(theCart))
		fmt.Println("Would you like to print the cart contents? This will FLOOD the screen if you choose yes. Answer with Y/n")
		var choice string
		fmt.Scanln(&choice)
		switch choice {
		case "y":
			fmt.Println(theCart)
		case "Y":
			fmt.Println(theCart)
		default:
			fmt.Println("Chosing not to print verbose output...")
		}
		startAnOrder()
	case "97":
		fmt.Println("Almost done! Would you like to perform a ban check on the accounts that are selected?")
		fmt.Println("Checking them will prevent users from getting banned accounts on arrival as well as stop 'i got an empty account' excuses")
		fmt.Println("Respond with Y/n")
		var check string
		fmt.Scanln(&check)
		switch check {
		case "y":
			checkCartAccounts()
		case "Y":
			checkCartAccounts()
		default:
			fmt.Println("Decided not to check accounts...")
		}
		fmt.Println("Okay, now just enter the customers name. This can be their usual name, discord tag, discord id, reddit name, or some other name that doesnt change")
		var theirname string
		fmt.Scanln(&theirname)
		fmt.Printf("%s eh? Fair enough. Added them to the logs for this order.\n", theirname)
		fmt.Println("Writing accounts to file and marking them as sold. This can take a bit for larger orders")
		for i := range theCart {
			theCart[i].Customer = theirname
			theCart[i].Sold = true
			theCart[i].SellDate = time.Now().Format(time.RFC1123)
		}
		writeCartToFile()
	case "98":
		theCart = nil
		theCart = make(map[int]*FileAccount)
	case "99":
		quit()
	default:
		fmt.Println("You entered something invalid. Try again.")
		startAnOrder()
	}
}

//checks accounts in the cart for bans or anything else
func checkCartAccounts() {

}

func addAccounts(item string, amt int) bool {
	//we got our item input so lets do a sanitization check just to make sure we got
	switch item { //we dont have to do anything other than make sure out switch case doesnt hit default
	case "decades":
	case "pixies":
	case "etherites":
	case "pierces":
	case "fairies":
	case "empty":
	default:
		return false
	}
	addedAmt := 0
	//loop through all accounts (O1)
	for i := 0; i < len(accounts); i++ {
		//if the account contains our desired item, then add it
		if accounts[i].ItemType == item && accounts[i].Sold == false && accounts[i].SellDate == "" && accounts[i].Customer == "" && addedAmt < amt { //redundant checks but oh well
			//theCart uses int i as key which should not ever duplicate. We don't give a damn about the key, just that the account is added properly
			//using this key also prevents adding other items to cart from interfering
			theCart[i] = accounts[i]
			addedAmt++
			//i shouldnt have to worry about adding customer name here as its a pointer
		}
	}
	if len(theCart) > amt { //we went through all the accounts but didnt get all the accounts we wanted
		return false
	}
	return true
}

func writeCartToFile() {
	fileName := "sales/" + getRandomString(12) + ".accounts"
	f, err := os.OpenFile(fileName, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		log.Println("Error opening order file:", err)
		f.Close()
	}
	defer f.Close()
	for i := range theCart {
		_, err = fmt.Fprintf(f, "%s:%s\n", theCart[i].Email, theCart[i].Password)
		if err != nil {
			fmt.Println("Error writing account:", err)
		}
	}
	fmt.Println("Succesfully wrote accounts to", fileName)
}

func printOrderMenu() {
	//fmt.Printf("\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n")
	fmt.Println("Welcome to the RealmSupply order menu!")
	fmt.Println("Select one of the options below to get started.")
	fmt.Println("Items already in cart will display below this message.")
	fmt.Printf("  1. Add Decade Rings\n")
	fmt.Printf("  2. Add Pixie Swords\n")
	fmt.Printf("  3. Add Etherite Daggers\n")
	fmt.Printf("  4. Add Pierce Spells\n")
	fmt.Printf("  5. Add Soulless Robes\n")
	fmt.Printf("  6. Add Fairy Plates\n")
	fmt.Printf("  7. Add Life Potions\n")
	fmt.Printf("  8. Add Defense Potions\n")
	fmt.Printf("  9. Add Empty Accounts (mules, daily login accounts, personal, ect)\n")
	fmt.Printf("\n")
	fmt.Printf("  96. Check Cart\n")
	fmt.Printf("  97. Finalize Order\n")
	fmt.Printf("  98. Empty Cart\n")
	fmt.Printf("  99. Exit Script\n")
	fmt.Printf("Accounts in cart: %d\n", len(theCart))
}

func checkProxies() {
	log.Printf("Checking %d proxies\n", len(proxyList))

	for i := 0; i < len(proxyList); i++ {
		fullString := fmt.Sprintf("%s%s%d", rootURL, listBit, GetTime())
		resp, err := getURL(fullString, i)
		// fmt.Println(fullString)
		if err != nil {
			//proxy is bad, obviously
			logProxyStatus(i, false, err)
			goto endcheck
		}
		if strings.Contains(resp, "<Chars") == true { //proxy works (within the timeout)
			logProxyStatus(i, true, nil)
		} else { //bad
			logProxyStatus(i, false, nil)
		}
	endcheck:
	}
	fmt.Println("All proxies checked. See proxies.log for details.")
}

func GetTime() int32 {
	return int32(time.Duration(time.Since(startTime) / time.Millisecond))
}

func logProxyStatus(i int, good bool, er error) {
	f, err := os.OpenFile("proxies.log", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		log.Println("Unable to open:", err)
	}
	switch good {
	case true:
		fmt.Fprintf(f, "%s | %s: Good\n", time.Now().String(), proxyList[i])
	case false:
		if er != nil {
			fmt.Fprintf(f, "%s | %s: Proxy issue | %s\n", time.Now().String(), proxyList[i], er.Error())
		} else {
			fmt.Fprintf(f, "%s | %s: Good, timeout\n", time.Now().String(), proxyList[i])
		}
	}
	f.Close()
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
	if ipHits >= dailyLimit {
		ipHits = 0
		proxyIndex++
		checkProxyBounds()
	}
}

func addProxyIP() string {
	return stripProxy(proxyList[proxyIndex])
}

func checkResponse(r string, i int) bool {
	//todo: depending on how many accounts, do an occasional flush to the vault.accounts to save progress
	//todo: some sort of "progress bar" status percentage whatever. Take the total to go thru divide by 4 add to array and do some checks
	accounts[i].LastStatusCheck = time.Now().Format(time.RFC1123)
	accounts[i].LastIP = addProxyIP()          //in the future we can try to do a "lookup" and use the same proxy for an account so we dont mix proxies and increase losses during a ban wave
	if strings.Contains(r, "<Chars") == true { //if we cant load the results then do some fallback checks
		//first lets group our "salesforce" data and see what pops up...
		sales1 := strings.Index(r, "<SalesForce>") + 12
		sales2 := strings.Index(r, "</SalesForce>")
		encodedData := r[sales1:sales2]
		rawData, err := base64.StdEncoding.DecodeString(encodedData)
		if err != nil {
			log.Printf("Error decoding %s base64 data. Error: ", accounts[i].Email)
			fmt.Println(err)
			//there will always be data in the salesforce part so we shouldnt have to worry about errors in here or returning
		}
		srawData := string(rawData)
		parsed, err := url.ParseQuery(srawData)
		if err != nil {
			log.Println("Couldn't decode values. Error:", err)
		}
		if parsed.Get("player_id") != "" { //means its been set already
			if accounts[i].LastIP != parsed.Get("player_id") {
				accounts[i].Comment = "ListerCode3"
			}
			//"date_joined" parameter could be useful, but the date format is not in any rfc format and i cant be arsed to custom parse it
		}
		if parsed.Get("player_name") != "" {
			if parsed.Get("player_name") != accounts[i].GameName { //set our ign if we haven't already or if it's mismatched
				accounts[i].GameName = parsed.Get("player_name")
			}
		}
		//lets check if we have a name... since unverified accounts can name themselves
		//if the account doesn't already have a name set then we will leave it for the namer script again
		if accounts[i].GameName == "" {
			//extract our name
			if parsed.Get("player_name") == "" {
				accounts[i].GameName = "nil"
			} else {
				accounts[i].GameName = parsed.Get("player_name")
			}
		}
		if strings.Contains(r, "VerifiedEmail") == true {
			accounts[i].Verified = true
		} else {
			accounts[i].Verified = false //not sure if I need an explicit false, but it might be handy since deca can "unverify" an account, although this power has never been demonstrated
		}
		if accounts[i].Verified == true { //we're verified so lets dig out some items!
			accounts[i].ItemType = "empty"                                         //we start with this and if items are detected, assign that item
			if strings.Contains(r, "2990,2990,2990,2990,2990,2990,2990") == true { //decades
				accounts[i].Filled = true
				accounts[i].FilledDate = time.Now().Format(time.RFC1123)
				accounts[i].ItemType = "decades"
			}
			if strings.Contains(r, "8608,8608,8608,8608,8608,8608,8608") == true { //etherites
				accounts[i].Filled = true
				accounts[i].FilledDate = time.Now().Format(time.RFC1123)
				accounts[i].ItemType = "etherites"
			}
			if strings.Contains(r, "9060,9060,9060,9060,9060,9060,9060") == true { //fairies
				accounts[i].Filled = true
				accounts[i].FilledDate = time.Now().Format(time.RFC1123)
				accounts[i].ItemType = "fairies"
			}
			if strings.Contains(r, "9063,9063,9063,9063,9063,9063,9063") == true { //pixies
				accounts[i].Filled = true
				accounts[i].FilledDate = time.Now().Format(time.RFC1123)
				accounts[i].ItemType = "pixies"
			}
		}
	} else { //run checks for accounts in use, account not exist, ect
		if strings.Contains(r, "Account in use") == true {
			//we have no clue if this account is going to dc at any moment so we'll have to skip it
			accounts[i].Comment = "ListerCode0" //thank god I included the "comment" field...
			return false
		}
		if strings.Contains(r, "credentials not valid") == true {
			//account must not exist...'
			accounts[i].Comment = "ListerCode1"
			return false
		}
		if strings.Contains(r, "please wait") == true {
			return true
		}

	}
	return false
}

func cleanUp() {
	saveAccounts() //flush any pending changes that may have been made
	log.Println("All finished with tasks!")
}

func saveAccounts() {
	file, err := os.OpenFile("vault.accounts", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		fmt.Printf("Unable to open %s\n", err)
		return
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.SetIndent("", "    ")
	err = enc.Encode(&accounts)
	if err != nil {
		fmt.Printf("error encoding config file: %s\n", err)
	}
}

//resetState flushes buffer, resets proxy states, and sleeps for t hours
func resetState(t int) {
	ipHits = 0
	proxyIndex = 0
	log.Printf("Sleeping for %d hours...\n", t)
	time.Sleep(time.Duration(t) * time.Hour)
}

func logError(app string, message string, item ...string) {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "%s | %s || %s: %s\n", time.Now().Format(time.RFC1123), app, item, message)
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
	defer panicSave()
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
		// myClient := &http.Client{}
		// trans := &http.Transport{}
		// trans.Proxy = http.ProxyURL(proxyURL)
		// myClient.Transport = trans
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

func readAccounts() {
	f, err := os.Open("vault.accounts")
	if err != nil {
		log.Printf("error opening vault file: %s\n", err)
		quit()
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&accounts); err != nil {
		fmt.Printf("error decoding config file: %s\n", err)
		quit()
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

func exportBannedAccounts() {
	f, err := os.OpenFile("banned_indices.txt", os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Println("Error writing banned accounts:", err)
		return
	}
	defer f.Close()
	b := new(bytes.Buffer)
	for i := 0; i < len(banList); i++ {
		fmt.Fprintf(b, "%d\n", banList[i])
	}
	_, err = f.Write(b.Bytes())
	if err != nil {
		fmt.Println("Error writing to banned list:", err)
		return //or quit instead?
	}
}

/*
	input: http://1.1.1.1:8085
	ouput: 1.1.1.1
*/
func stripProxy(s string) string {
	var z string
	y := strings.Split(s, ":")[1]
	z = y[2:]
	return z
}

func panicSave() {
	if err := recover(); err != nil {
		fmt.Println("Saving from crash")
		saveAccounts()
	}
}

func getRandomHeader() string {
	return userAgents[rand.Int31n(int32(len(userAgents)))]
}

func sigHandler(c chan os.Signal) {
	signal := <-c
	_ = signal
	fmt.Printf("\nGot signal. Shutting down...\n")
	saveAccounts()
	os.Exit(0)
}

func quit() {
	os.Exit(0)
}

//BotAccount is the format used by the clientless bots
type BotAccount struct {
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
