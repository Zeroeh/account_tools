package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

/*
	account_muledump - project turns email:password formats to muledump accounts.js format
		useful for resellers and lesser rwt kids who dont have swag tools like me
*/

const (
	saveFileMsg   = "Enter the name of the file to save the accounts to. Example: new.accounts"
	readFileMsg   = "Enter the name of the file to scan. Example: items.accounts\n The file must be in the same directory as this exe."
	quitDlg       = "Exiting in a few seconds..."
	quitSleepTime = 10
)

var (
	oldAccounts map[string]string
)

func main() {
	oldAccounts = make(map[string]string)
	fmt.Println(readFileMsg)
	var choice string
	fmt.Scanln(&choice)
	readAccounts(choice)
	fmt.Println(saveFileMsg)
	var choice2 string
	fmt.Scanln(&choice2)
	saveAccounts(choice2)
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

func saveAccounts(fname string) {
	f, err := os.OpenFile(fname, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		fmt.Println("Error creating new file:", err)
		quit()
	}
	for i, v := range oldAccounts {
		email := i
		password := v
		fmt.Fprintf(f, "'%s':'%s',\r\n", email, password)
	}
	f.Close()
}

func quit() {
	fmt.Println(quitDlg)
	time.Sleep(time.Second * quitSleepTime)
	os.Exit(0)
}
