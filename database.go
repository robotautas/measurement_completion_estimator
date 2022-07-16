package main

import (
	"fmt"
	"log"
	"net/url"
	"time"

	client "github.com/influxdata/influxdb1-client"
)

var con = getDBConnection()

// Returns a database connection
func getDBConnection() *client.Client {
	host, err := url.Parse(fmt.Sprintf("http://%s:%d", "localhost", 8086))
	check(err)
	conf := client.Config{
		URL: *host,
	}
	con, err := client.NewClient(conf)
	check(err)
	return con
}

// write transformed outputs from arduino to database
func writeLineToDatabase(con *client.Client, output map[string]interface{}) {
	pt := client.Point{
		Measurement: "etc",
		Fields:      output,
		Time:        time.Now()}
	pts := []client.Point{pt}
	bp := client.BatchPoints{
		Points:          pts,
		Database:        "ssams",
		RetentionPolicy: "my_policy", // pabandyti koreguoti.
	}
	_, err := con.Write(bp)
	if err != nil {
		log.Fatal(err)
	}
}

func check(err error) {
	if err != nil {
		panic(err.Error())
	}
}
