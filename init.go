package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var (
	runOnAws   bool
	syncerMode bool
)

func init() {
	flag.IntVar(&listenPort, "port", 9000, "listen port")
	flag.Parse()
	fmt.Println("Listen Port : ", listenPort)

	if az := getEC2MetaData("availability-zone"); az != "" {
		runOnAws = true
		store.host.AZ = az
	}
	region := getEC2MetaData("region")
	vpcID := getEC2MetaData("vpc-id")
	fmt.Printf("%v\n", getEC2IPList(region, vpcID))
}

func getEC2MetaData(field string) (value string) {
	sess := session.Must(session.NewSession())
	svc := ec2metadata.New(sess)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if svc.AvailableWithContext(ctx) {
		doc, _ := svc.GetInstanceIdentityDocumentWithContext(ctx)
		switch field {
		case "availability-zone":
			value = doc.AvailabilityZone
		case "region":
			value = doc.Region
		case "vpc-id":
			mac, _ := svc.GetMetadataWithContext(ctx, "/mac")
			value, _ = svc.GetMetadataWithContext(ctx, "/network/interfaces/macs/"+mac+"/vpc-id")
		default:
			if strings.Index(field, "/") == -1 {
				value, _ = svc.GetMetadataWithContext(ctx, "/"+field)
			} else {
				value, _ = svc.GetMetadataWithContext(ctx, field)
			}
		}
	}
	return
}

func getEC2IPList(region, vpcID string) []string {
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
		return []string{}
	}
	ipList := []string{}
	for _, ni := range result.NetworkInterfaces {
		if canConnect(*ni.PrivateIpAddress) {
			ipList = append(ipList, *ni.PrivateIpAddress)
		}
	}
	return ipList
}

func canConnect(ip string) bool {
	conn, err := net.DialTimeout("tcp", ip+":"+strconv.Itoa(listenPort), time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}
