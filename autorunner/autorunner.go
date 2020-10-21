package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
)

/* autorunner.go - Auto run the bots and increment the account index once a certain amount of bots have disconnected */

var (
	settings     *Settings
	next         = make(chan int)
	countMap     = make(map[int]int)
	badBatch     = false
	resetTrigger = false
)

func main() {
	readSettings()
	if settings.UseNotifier == true {
		go controlListener()
		for {
			if settings.Index >= settings.ConnLimit {
				log.Println("Reached the bot limit, quitting...")
				quit()
			}
			log.Println("Launching program...")
			cmd := exec.Command("./bots")
			cmd.Start()
			log.Println("Waiting for signal...")
			select { //block until we get the "ok" to restart
			case <-next:
				cmd.Process.Kill()
				cmd.Wait() //prevent zombie processes
				log.Println("Signal received!")
				if badBatch == true && resetTrigger == false {
					resetTrigger = true
				}
			}
		}
	} else {
		fmt.Println("UseNotifier in settings.json is disabled. Stopping...")
		quit()
	}
}

func controlListener() {
	listener, err := net.Listen("tcp", "127.0.0.1:6661")
	if err != nil {
		fmt.Println("Error starting listener:", err)
		quit()
	}
	for {
		client, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting client:", err)
		}
		// log.Println("Accepted client!")
		go handleConn(client)
	}
}

func handleConn(c net.Conn) {
	readBuf := new(bytes.Buffer)
	count, err := readBuf.ReadFrom(c)
	if err != nil { //nil or io.EOF????
		fmt.Println("Got err on readFrom:", err)
	}
	defer c.Close()
	if count > 0 {
		accountsConnected := int(readBuf.Bytes()[0])
		if accountsConnected >= 17 { //i find this to be the sweet spot between accounts completed and time spent running the bots
			log.Println("Current bots reported:", accountsConnected)
			countMap[3] = countMap[2]
			countMap[2] = countMap[1]
			countMap[1] = countMap[0]
			countMap[0] = accountsConnected
			//four checks should be enough
			if countMap[0] == countMap[1] && countMap[1] == countMap[2] && countMap[2] == countMap[3] {
				if badBatch == true {
					badBatch = false
					log.Println("Bots got messed up twice... Moving to the next 100.")
					settings.Index += 100
					writeSettings()
					next <- 0
					return
				}
				log.Println("Bots could be messed up... restarting them")
				badBatch = true
				next <- 0 //force relaunch
			}
		} else { //we got enough, start the next batch
			// log.Println("Bot count is low enough, writing settings and signaling process...")
			settings.Index += 100
			writeSettings()
			next <- 0
		}
	} else {
		fmt.Println("Got 0 from read?")
	}
}

func readSettings() {
	file, err := os.Open("config/settings.json")
	if err != nil {
		log.Println("Error opening settings for read:", err)
		quit()
	}
	defer file.Close()
	err = json.NewDecoder(file).Decode(&settings)
	if err != nil {
		log.Println("Error decoding settings:", err)
		quit()
	}
}

func writeSettings() {
	file, err := os.OpenFile("config/settings.json", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Println("Error opening settings for write:", err)
		fmt.Println("Current index is", settings.Index)
		return //we can try again later
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.SetIndent("", "    ")
	err = enc.Encode(&settings)
	if err != nil {
		log.Println("Error encoding settings:", err)
		return //try again later
	}
}

type Settings struct {
	Amount       int    `json:"amount"`
	Index        int    `json:"index"`
	ConnLimit    int    `json:"connlimit"`
	WaitPeriodMS int    `json:"waitperiodms"`
	GameVersion  string `json:"gameVersion"`
	ProfileCPU   bool   `json:"profilecpu"`
	ThreadDelay  int    `json:"threaddelay"`
	ReconDelay   int    `json:"recondelay"`
	SaveDelay    int    `json:"savedelay"`
	ReceiveItem  int    `json:"receiveitem"`
	UseNotifier  bool   `json:"usenotifier"`
}

func quit() {
	os.Exit(1)
}
