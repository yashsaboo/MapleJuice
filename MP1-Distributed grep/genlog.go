package main

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"
)

// Strings for genlog functions to access
// Frequent 50%, somewhat frequent 15%, rare 2%, random the rest
var freqStrings = []string{"[Keynes] But this long run is a misleading guide to current affairs. In the long run we are all dead.",
	"[Keynes] The old saying holds. Owe your banker £1000 and you are at his mercy; owe him £1 million and the position is reversed.",
	"[Keynes] When my information changes, I alter my conclusions. What do you do, sir?"}

var someWhatFreqStrings = []string{"[Gandhi] Be the change that you want to see in the world.",
	"[Gandhi] A man is but a product of his thoughts. What he thinks he becomes.",
	"[Gandhi] First they ignore you, then they laugh at you, then they fight you, then you win."}

var rareStrings = []string{"[Bismarck] Not by speeches and votes of the majority, are the great questions of the time decided — that was the error of 1848 and 1849 — but by iron and blood.",
	"[Bismarck] We Germans fear God, but nothing else in the world."}

// Strings to be present in all/some/one files
var strForAllFiles = "[Hagel] We learn from history that we do not learn from history."

var strForSomeFiles = "[T.R.] Speak softly and carry a big stick; you will go far."

var strForOneFile = "[King jr.] Darkness cannot drive out darkness; only light can do that. Hate cannot drive out hate; only love can do that."

// GenLog generate random and known log strings in the form of list of strings
func genLog(machineNo int, lineCnt int, fileSize int) []string {

	// Create log based on total line counts
	if lineCnt >= 0 {
		return []string{genLogLineCnt(machineNo, lineCnt)}
	}

	// Create log based on given file size
	// Due to it somehow becoming increasingly slow when the log is large,
	// do this iteratively
	curSize := 0
	const increSize = 1000000
	var resultStore []string
	for curSize+increSize < fileSize {
		resultStore = append(resultStore, genLogFileSize(increSize))
		curSize += len(resultStore[len(resultStore)-1])
	}

	if (fileSize - curSize) > 0 {
		resultStore = append(resultStore, genLogFileSize(fileSize-curSize))
	}

	return resultStore
}

// genRandAlphaNumString generate random alphanumeric strings wth given length
func genRandAlphaNumString(length int) string {

	AlphaNumericChar := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

	result := ""
	for i := 0; i < length; i++ {
		result += string(AlphaNumericChar[rand.Intn(len(AlphaNumericChar))])
	}

	return result
}

// specialString is the struct where we store the special string and the line number
// that it will be placed in the log outputs
type specialString struct {
	lineNo int
	spStr  string
}

// getNewNotDupRandInt generates none duplicate integers in range [0, rng)
func getNewNotDupRandInt(rng int, target *map[int]bool) int {

	result := rand.Intn(rng)

	for (*target)[result] {
		result = rand.Intn(rng)
	}

	(*target)[result] = true

	return result
}

// genLogLineCnt generates log based on given line count
func genLogLineCnt(machineNo int, lineCnt int) string {
	// Decide whether to add special lines and if that is is the case, where to place them
	specialLines := []specialString{}
	alreadyTaken := make(map[int]bool)
	if lineCnt < 100 {
		fmt.Println("Too less lines, omitting special log lines")
	} else {

		// Special line that will only appear on log of machine 3
		// Note that machine numbers start from 0
		if machineNo == 3 {
			specialLines = append(specialLines,
				specialString{lineNo: getNewNotDupRandInt(lineCnt, &alreadyTaken),
					spStr: strForOneFile})
		}

		// Special line that will only appear on log of odd numbered machine
		if machineNo%2 == 1 {
			specialLines = append(specialLines,
				specialString{lineNo: getNewNotDupRandInt(lineCnt, &alreadyTaken),
					spStr: strForSomeFiles})
		}

		// Special line that will appear on all log files
		specialLines = append(specialLines,
			specialString{lineNo: getNewNotDupRandInt(lineCnt, &alreadyTaken),
				spStr: strForAllFiles})
	}

	// Start generating resulting log entries
	result := ""
	for i := 0; i < lineCnt; i++ {

		// Encountered special line line count, store special line
		if alreadyTaken[i] {
			for _, record := range specialLines {
				if record.lineNo == i {
					result += record.spStr + "\n"
					break
				}
			}

			continue
		}

		// Frequent 100~50%; somewhat frequent 50~35%; rare 35%~33%; the rest are just random
		randNum := rand.Float64()
		if randNum >= 0.5 {
			result += freqStrings[rand.Intn(len(freqStrings))] + "\n"
		} else if 0.5 > randNum && randNum >= 0.35 {
			result += someWhatFreqStrings[rand.Intn(len(someWhatFreqStrings))] + "\n"
		} else if 0.35 > randNum && randNum >= 0.33 {
			result += rareStrings[rand.Intn(len(rareStrings))] + "\n"
		} else {
			result += "[Monkey] " + genRandAlphaNumString(rand.Intn(50)+50) + "\n"
		}

	}

	return result
}

// genLogFileSize generate log file based on filesize, may not be exact
func genLogFileSize(fileSize int) string {
	result := ""
	oldCnt := 0
	for len(result) < fileSize {
		if len(result)/10000 > oldCnt {
			fmt.Println(len(result))
			oldCnt = len(result) / 10000
		}

		// Frequent 100~50%; somewhat frequent 50~35%; rare 35%~33%; the rest are just random
		randNum := rand.Float64()
		if randNum >= 0.5 {
			result += freqStrings[rand.Intn(len(freqStrings))] + "\n"
		} else if 0.5 > randNum && randNum >= 0.35 {
			result += someWhatFreqStrings[rand.Intn(len(someWhatFreqStrings))] + "\n"
		} else if 0.35 > randNum && randNum >= 0.33 {
			result += rareStrings[rand.Intn(len(rareStrings))] + "\n"
		} else {
			result += "[Monkey] " + genRandAlphaNumString(rand.Intn(50)+50) + "\n"
		}

	}

	return result
}

func main() {

	// check OS arguments
	if len(os.Args) != 5 && len(os.Args) != 4 {
		fmt.Println("Usage: go run genlog.go MACHINE_NO LOG_PATH LINE_CNT FILE_SIZE")
		fmt.Println("Generate log file via line count or via file size")
		fmt.Println("To generate log file via line count, pass >= 0 value in LINE_CNT")
		fmt.Println("To generate log file via file size, pass < 0 value in LINE_CNT and pass target FILE_SIZE")
		return
	}

	// Convert arguments to int
	machineNo, _ := strconv.Atoi(os.Args[1])
	lineCnt, _ := strconv.Atoi(os.Args[3])
	var fileSize int
	if len(os.Args) == 5 {
		fileSize, _ = strconv.Atoi(os.Args[4])
	} else {
		fileSize = -1
	}

	// Validate arguments
	if lineCnt < 0 && fileSize < 0 {
		fmt.Println("Usage: go run genlog.go MACHINE_NO LOG_PATH LINE_CNT FILE_SIZE")
		fmt.Println("Generate log file via line count or via file size")
		fmt.Println("To generate log file via line count, pass >= 0 value in LINE_CNT")
		fmt.Println("To generate log file via file size, pass < 0 value in LINE_CNT and pass target FILE_SIZE (>= 0)")
		return
	}

	// Seed random function
	rand.Seed(time.Now().UTC().UnixNano())

	// Start generating log
	result := genLog(machineNo, lineCnt, fileSize)

	// Create target log file to write
	fp, err := os.Create(os.Args[2])
	if err != nil {
		fmt.Printf("Cannot create file %s\n", os.Args[2])
		return
	}
	defer fp.Close()

	// Write generated log to file
	for _, str := range result {
		n, err := fp.WriteString(str)
		if n != len(str) || err != nil {
			fmt.Printf("Cannot write to file %s\n", os.Args[2])
			return
		}
	}

	fmt.Println("Done")
}
