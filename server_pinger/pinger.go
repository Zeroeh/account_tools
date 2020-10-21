package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

type Server struct {
	Name     string
	IP       string
	PingTime float64
}

var (
	serverList []Server
)

const (
	rootURL = "https://realmofthemadgodhrd.appspot.com/char/list?guid=268347962398469824629846789"
)

func main() {
	fmt.Println("Grabbing server list...")
	xmlData, err := getURL(rootURL)
	if err != nil {
		fmt.Println("Error grabbing server list:", err)
		fmt.Scanln()
	}
	index1 := strings.Index(xmlData, "<Servers>") + 9
	index2 := strings.Index(xmlData, "</Servers>")
	if index1 == -1 || index2 == -1 {
		fmt.Println("Bad indices. Web output:", xmlData)
		fmt.Scanln()
	}
	serverXML := xmlData[index1:index2]
	amt := strings.Count(serverXML, "<Server>")
	fmt.Printf("Grabbed %d servers.\n", amt)
	serverList = make([]Server, amt+1)
	servers := strings.Split(serverXML, "<Server>")
	for i := range servers {
		if i == 0 { //some weird empty shit is getting caught, skip over it
			continue
		} else {
			name1 := strings.Index(servers[i], "<Name>") + 6
			name2 := strings.Index(servers[i], "</Name>")
			name := servers[i][name1:name2]
			serverList[i].Name = name
			ip1 := strings.Index(servers[i], "<DNS>") + 5
			ip2 := strings.Index(servers[i], "</DNS>")
			serverList[i].IP = servers[i][ip1:ip2]
		}

	}
	for i := 0; i < len(serverList); i++ {
		if serverList[i].IP == "" {
			continue
		} else {
			go serverList[i].pingRotMGServer()
		}
	}
	fmt.Println("Please wait about 10 seconds for results to display. The best servers will be closer to the top of the list.")
	fmt.Scanln()
}

func (s *Server) pingRotMGServer() {
	buffer := make([]byte, 1)
	conn, err := net.Dial("tcp", s.IP+":2050")
	if err != nil {
		fmt.Printf("Error dialing %s(%s): ", s.Name, s.IP)
		fmt.Println(err)
		return
	}
	start := time.Now()
	writeBuffer := []byte("\x00\x00\x00\x4e\x1e\xff\xff\xff")
	amt, err := conn.Write(writeBuffer)
	if err != nil {
		fmt.Printf("Error writing %s(%s): ", s.Name, s.IP)
		fmt.Println(err)
		return
	}
	if amt != len(writeBuffer) {
		fmt.Println("Didn't write 1 byte, wrote", amt)
	}
	amt, err = conn.Read(buffer)
	if err != nil {
		fmt.Printf("Error reading %s(%s): ", s.Name, s.IP)
		fmt.Println(err)
		return
	}
	if amt != 1 {
		fmt.Println("Didn't read 1 byte, read", amt)
	}
	elapsed := time.Since(start)
	s.PingTime = (elapsed.Seconds() - 10) * 1000
	fmt.Printf("RTT for %s is %f ms\n", s.Name, s.PingTime)
}

func getURL(s string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, s, nil)
	if err != nil {
		return "", err
	}
	myClient := &http.Client{}
	ex, err := myClient.Do(req)
	if err != nil {
		return "", err
	}
	body, _ := ioutil.ReadAll(ex.Body)
	ex.Body.Close()
	return string(body), err
}

func quit() {
	os.Exit(0)
}
