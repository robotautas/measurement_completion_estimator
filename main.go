package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// const runDuration int = 198 replacen by a dynamic function
const postMeasurementRuntime int = 25

func main() {
	for {
		completionTime, sleepTime := getCompletionTime()
		preparedValues := prepareValues(completionTime)
		writeLineToDatabase(con, preparedValues)
		fmt.Printf("%v: ETC -> %v, ETL -> %v\n", time.Now(), preparedValues["etc"], preparedValues["etl"])
		time.Sleep(time.Second * time.Duration(sleepTime))
	}
}

func getRunDuration() (int, error) {
	url := "http://172.16.176.40/csquery.php?act=query&list=DOSEstlog"
	resp, err := http.Get(url)
	if err != nil {
		return -1, fmt.Errorf("error fetching URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return -1, fmt.Errorf("error: status code %d", resp.StatusCode)
	}

	html, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return -1, fmt.Errorf("error reading response body: %w", err)
	}

	lines := strings.Split(string(html), "CntDwnC")
	cyclesStr := strings.Fields(lines[1])
	cycles, err := strconv.Atoi(cyclesStr[0])

	if err != nil {
		return -1, fmt.Errorf("error converting cycles to int: %w", err)
	}

	seconds := (cycles + 100) * 115 / 1000
	return seconds, nil
}

func getCompletionTime() (string, int) {
	runDuration, err := getRunDuration()
	if err != nil {
		return "failed to read run duration", 300
	}

	url := "http://172.16.176.40/csquery.php?act=dose&list=item"
	resp, err := http.Get(url)

	if err != nil {
		return "Error: bad response", runDuration
	}

	defer resp.Body.Close()

	html, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return "Err: can't read response body", runDuration
	}

	// in case server is down..
	if !strings.Contains(string(html), "<pre>") {
		return "No data!", runDuration
	}

	//get <pre> element
	pre := strings.Split(string(html), "<pre>")[1]
	pre = strings.Split(pre, "</pre>")[0]
	lines := strings.Split(pre, "\n")

	var totalSeconds int64 = 0

	dataLineCounter := 0

	runsIndex, compIndex := getRunsCompIndex(lines)

	for _, line := range lines {
		// split line by whitespace
		lineValues := strings.Fields(line)
		if len(lineValues) > 0 && lineValues[0] == "_" {
			dataLineCounter++
			runsValue, err := strconv.Atoi(lineValues[runsIndex])
			if err != nil {
				return "Runs Value Error", runDuration
			}
			compValue, err := strconv.Atoi(lineValues[compIndex])
			if err != nil {
				return "Comp Value Error", runDuration
			}
			// if all runs are not complete ...
			if runsValue != compValue {
				seconds := int64((runsValue - compValue) * runDuration)
				totalSeconds += seconds
			}
		}
	}

	if totalSeconds == 0 {
		return "Complete", runDuration
	}

	// Post measurement time, when the wheel turns to initial position.
	afterTime := dataLineCounter * postMeasurementRuntime
	totalSeconds += int64(afterTime)

	timeNow := time.Now()
	timeComplete := timeNow.Add(time.Second * time.Duration(totalSeconds))
	etc := timeComplete.Format("2006-01-02 15:04:05")

	dur := time.Duration(time.Second * time.Duration(totalSeconds)).String()
	var etl string

	if strings.Contains(dur, "m") {
		etl = fmt.Sprintf("%vm", strings.Split(dur, "m")[0])
	} else {
		etl = dur
	}

	return etc + "|" + etl, runDuration
}

// get Runs and Comp columns indexes dinamically
func getRunsCompIndex(lines []string) (int, int) {
	var columnNames []string

	for _, line := range lines {
		if strings.HasPrefix(line, "E") {
			columnNames = strings.Fields(line)
			break
		}
	}

	var runsIndex int
	var compIndex int
	var doubleNames int

	for i, name := range columnNames {
		// check for two-word names ("Sample Name" and "Sample Name2")
		if name == "Sample" {
			doubleNames++
		}
		if name == "Runs" {
			runsIndex = i - doubleNames
		} else if name == "Comp" {
			compIndex = i - doubleNames
		}
	}

	return runsIndex, compIndex
}

func prepareValues(s string) map[string]interface{} {
	preparedValues := make(map[string]interface{})
	if strings.Contains(s, "|") {
		splitted := strings.Split(s, "|")
		preparedValues["etc"] = splitted[0]
		preparedValues["etl"] = splitted[1]
	} else {
		preparedValues["etc"] = s
		preparedValues["etl"] = s
	}
	return preparedValues
}
