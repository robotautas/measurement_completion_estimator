package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
)

var pool = newPool()

func newPool() *redis.Pool {
	return &redis.Pool{
		MaxIdle:   80,
		MaxActive: 12000,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", ":6379")
			if err != nil {
				panic(err.Error())
			}
			return c, err
		},
	}
}

func saveToRedis(completionTime string) {
	client := pool.Get()
	defer client.Close()
	_, err := client.Do("SET", "completionTime", completionTime)
	if err != nil {
		panic(err)
	}
}

func main() {
	for {
		saveToRedis(getCompletionTime())
		time.Sleep(time.Minute * 6)
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
		return "Err: can't read body"
	}

	//get <pre> element
	pre := strings.Split(string(html), "<pre>")[1]
	pre = strings.Split(pre, "</pre>")[0]
	lines := strings.Split(pre, "\n")

	var totalSeconds int64 = 0

	for _, line := range lines {
		// split line by whitespace
		lineValues := strings.Fields(line)
		if len(lineValues) > 0 && lineValues[0] == "_" {
			// get Runs and Comp columns values
			runsValue, err := strconv.Atoi(lineValues[7])
			if err != nil {
				return "Runs Value Error"
			}
			compValue, err := strconv.Atoi(lineValues[8])
			if err != nil {
				return "Comp Value Error"
			}
			fmt.Printf("%v   %v\n", runsValue, compValue)

			// if all runs are not complete ...
			if runsValue != compValue {
				seconds := int64((runsValue-compValue)*180 + 20)
				totalSeconds += seconds
				// fmt.Printf("secs: %v\n", seconds)
			}
		}
	}

	timeNow := time.Now()
	timeComplete := timeNow.Add(time.Second * time.Duration(totalSeconds))
	returnString := timeComplete.Format("2006-01-02 15:04:05")

	return returnString
}
