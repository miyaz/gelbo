package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

var (
	runOnAws bool
)

func init() {
	flag.IntVar(&idleTimeout, "timeout", 65, "idle timeout")
	flag.Parse()
	fmt.Printf("Idle Timeout: %d sec\n\n", idleTimeout)

	if az := getEC2MetaData("availability-zone"); az != "" {
		fmt.Println("running on AWS")
		runOnAws = true
		store.host.AZ = az
	} else {
		fmt.Println("detected running on non-AWS")
	}
}

func getFromIMDS(path string) (data string) {
	// can not access imds from docker container when use aws-sdk-go/aws/ec2metadata
	addr := "http://169.254.169.254/latest/meta-data"
	client := http.Client{
		Timeout: time.Second,
	}
	resp, err := client.Get(addr + path)
	if err != nil {
		return ""
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	return string(body)
}

func getEC2MetaData(field string) (value string) {
	//169.254.169.254/latest/meta-data/placement/availability-zone
	az := getFromIMDS("/placement/availability-zone")
	if az != "" {
		switch field {
		case "availability-zone":
			value = az
		case "region":
			value = az[:len(az)-1]
		case "vpc-id":
			mac := getFromIMDS("/mac")
			value = getFromIMDS("/network/interfaces/macs/" + mac + "/vpc-id")
		default:
			if strings.Index(field, "/") == -1 {
				value = getFromIMDS("/" + field)
			} else {
				value = getFromIMDS(field)
			}
		}
	}
	return
}
