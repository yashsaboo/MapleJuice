package main

import (
	"bufio"
	"log"
	"regexp"
	"strings"

	// "encoding/json"
	"fmt"
	// "log"
	// "net"
	"os"
	// "strings"
)

var m map[string]int

func trimWhitespaceAndNewlineFeedFromString(str string) string {
	s := strings.Replace(str, "\n", "", -1)
	s = strings.TrimSpace(s)
	return s
}

//https://golangcode.com/how-to-remove-all-non-alphanumerical-characters-from-a-string/
func removeAllNonAlphaNumericCharactersFromString(str string) string {
	// Make a Regex to say we only want letters, space and numbers
	reg, err := regexp.Compile("[^a-zA-Z0-9 ]+")
	if err != nil {
		log.Fatal(err)
	}
	return reg.ReplaceAllString(str, "")
}

//For reference: https://blog.golang.org/go-maps-in-action
func translateToHashMap(path string) {
	file, err := os.Open(path)
	if err != nil {
		fmt.Print("Couldn't open MapInputFile because")
		fmt.Println(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		line = trimWhitespaceAndNewlineFeedFromString(line)
		line = removeAllNonAlphaNumericCharactersFromString(line)
		s := strings.Split(line, " ")

		for _, word := range s {
			if word == "" {
				break
			}
			i, ok := m[word] //Checks for the word in hashmap. If present, then i stores the current value and ok holds true bool value, else, false value and i=0
			if ok == true {
				m[word] = i + 1 //If value already present, then just increment the count
			} else {
				m[word] = 1 //If value not present, then initilialise it to 1
			}
		}
	}
}

// Main thread of execution
// go run hashmap.go
func main() {

	//Create a hashmap
	m = make(map[string]int)

	translateToHashMap("shared/inputData/wordCountInput.txt")

	for key, value := range m {
		fmt.Println("Key:", key, "Value:", value)
	}

}
