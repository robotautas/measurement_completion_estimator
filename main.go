package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const runDuration int = 198
const postMeasurementRuntime int = 25

func main() {
	for {
		start := time.Now()
		completionTime := getCompletionTime()
		preparedValues := prepareValues(completionTime)
		fmt.Printf("prepared values : %v\n", preparedValues)
		writeLineToDatabase(con, preparedValues)
		fmt.Printf("%v: ETC -> %v, ETL -> %v\n", time.Now(), preparedValues["etc"], preparedValues["etl"])
		fmt.Println("runtime: ", time.Since(start))
		time.Sleep(time.Second * time.Duration(runDuration))
	}
}

func getCompletionTime() string {
	url := "http://172.16.176.40/csquery.php?act=dose&list=item"
	resp, err := http.Get(url)

	if err != nil {
		return "Error: bad response"
	}

	defer resp.Body.Close()

	html, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return "Err: can't read response body"
	}

	//get <pre> element
	pre := strings.Split(string(html), "<pre>")[1]
	pre = strings.Split(pre, "</pre>")[0]
	lines := strings.Split(pre, "\n")

	var totalSeconds int64 = 0

	dataLineCounter := 0

	for _, line := range lines {
		// split line by whitespace
		lineValues := strings.Fields(line)
		if len(lineValues) > 0 && lineValues[0] == "_" {
			dataLineCounter++
			runsValue, err := strconv.Atoi(lineValues[7])
			if err != nil {
				return "Runs Value Error"
			}
			compValue, err := strconv.Atoi(lineValues[8])
			if err != nil {
				return "Comp Value Error"
			}
			// if all runs are not complete ...
			if runsValue != compValue {
				seconds := int64((runsValue - compValue) * runDuration)
				totalSeconds += seconds
			}
		}
	}

	if totalSeconds == 0 {
		return "Complete"
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

	return etc + "|" + etl
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
