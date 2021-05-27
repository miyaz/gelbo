package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var (
	runOnAws   bool
	syncerMode bool
)

func init() {
	flag.IntVar(&listenPort, "listen", 80, "listen port")
	flag.IntVar(&syncerPort, "syncer", 80, "syncer port")
	flag.IntVar(&idleTimeout, "timeout", 65, "idle timeout")
	flag.Parse()
	fmt.Printf("Listen Port : %d\n", listenPort)
	fmt.Printf("Syncer Port : %d\n", syncerPort)
	fmt.Printf("Idle Timeout: %d sec\n\n", idleTimeout)

	if az := getEC2MetaData("availability-zone"); az != "" {
		fmt.Println("running on AWS")
		runOnAws = true
		store.host.AZ = az
		ec2IpList, err := getEC2IPList()
		if err != nil {
			fmt.Println("ec2.DescribeNetworkInterfaces is not allowed, so turn off syncerMode.")
		} else {
			syncerMode = true
			fmt.Println("running in syncer mode(port:" + strconv.Itoa(syncerPort) + ")")
			fmt.Printf("%v\n", ec2IpList)
		}
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

func getEC2IPList() ([]string, error) {
	region := getEC2MetaData("region")
	vpcID := getEC2MetaData("vpc-id")
	config := &aws.Config{
		Region: aws.String(region),
	}
	sess := session.Must(session.NewSession(config))
	ec2Svc := ec2.New(sess)
	params := &ec2.DescribeNetworkInterfacesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("vpc-id"),
				Values: []*string{aws.String(vpcID)},
			},
			&ec2.Filter{
				Name:   aws.String("status"),
				Values: []*string{aws.String("in-use")},
			},
		},
	}
	result, err := ec2Svc.DescribeNetworkInterfaces(params)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return nil, err
	}
	ipList := []string{}
	for _, ni := range result.NetworkInterfaces {
		if !strings.HasPrefix(*ni.Description, "ELB") {
			ipList = append(ipList, *ni.PrivateIpAddress)
		}
	}
	return ipList, nil
}
