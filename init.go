package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

var (
	runOnAws bool
)

func init() {
	flag.BoolVar(&noLog, "nolog", false, "disable access logging")
	flag.IntVar(&httpPort, "http", 80, "http port")
	flag.IntVar(&httpsPort, "https", 443, "https port")
	flag.IntVar(&idleTimeout, "timeout", 65, "idle timeout")
	flag.Parse()
	zlog := zerolog.New(os.Stderr).Level(zerolog.DebugLevel).With().
		Int("http", httpPort).
		Int("https", httpsPort).
		Int("timeout", idleTimeout).
		Bool("nolog", noLog).Logger()

	if metaDataType := getMetaDataType(); metaDataType != "" {
		zlog.Log().Msg("running on AWS")
		runOnAws = true
		store.host.AZ = getAZ(metaDataType)
		if metaDataType == "ec2" {
			store.host.InstanceType = getEC2MetaData("instance-type")
		}
	} else {
		zlog.Log().Msg("running on non-AWS")
	}
}

func getMetaDataType() string {
	client := http.Client{
		Timeout: time.Second,
	}
	if az := getFromIMDS("/placement/availability-zone"); az != "" {
		return "ec2"
	} else if endpoint := os.Getenv("ECS_CONTAINER_METADATA_URI_V4"); endpoint != "" {
		_, err := client.Get(endpoint)
		if err != nil {
			return ""
		}
		return "ecs"
	}
	return ""
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

// for unmarshal from {ECS_CONTAINER_METADATA_URI_V4}/task
type MetadataTask struct {
	//Cluster          string
	//TaskARN          string
	AvailabilityZone string
}

func getContainerMetadata(field string) string {
	// now not using field variable
	// TODO: get a value specified by field variable

	addr := os.Getenv("ECS_CONTAINER_METADATA_URI_V4") + "/task"
	client := http.Client{
		Timeout: time.Second,
	}
	resp, err := client.Get(addr)
	if err != nil {
		return ""
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	var md MetadataTask
	if err := json.Unmarshal(body, &md); err != nil {
		return ""
	}

	return string(md.AvailabilityZone)
}

func getAZ(metaDataType string) string {
	if metaDataType == "ec2" {
		return getEC2MetaData("availability-zone")
	} else if metaDataType == "ecs" {
		return getContainerMetadata("availability-zone")
	}
	return ""
}
