package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

var (
	runOnEC2 bool
)

func init() {
	flag.IntVar(&httpPort, "http", 80, "http port")
	flag.IntVar(&httpsPort, "https", 443, "https port")
	flag.IntVar(&grpcPort, "grpc", 50051, "grpc port")
	flag.IntVar(&grpcsPort, "grpcs", 50052, "grpcs ([s] means over tls) port")
	flag.IntVar(&idleTimeout, "timeout", 65, "idle timeout. if 0 is specified, no limit")
	flag.IntVar(&probeInterval, "interval", 15, "tcp-keepalive probe interval. if 0 is specified, probe is not sent")
	flag.IntVar(&grpcInterval, "grpcping", 30, "grpc ping frame interval. if 0 is specified, ping frame is not sent")
	flag.Int64Var(&wsInterval, "wsping", 30, "websocket ping interval")
	//flag.Int64Var(&maxMessageSize, "wsmaxsize", 1024, "websocket max message size")
	flag.BoolVar(&execFlag, "exec", false, "enable exec feature")
	flag.BoolVar(&proxyFlag, "proxy", false, "enable proxy protocol")
	flag.BoolVar(&noLogFlag, "nolog", false, "disable access logging")
	flag.Parse()
	if probeInterval < 0 {
		fmt.Printf("invalid value \"%d\" for flag -interval: less than zero\n", probeInterval)
		os.Exit(2)
	}
	if grpcInterval < 0 {
		fmt.Printf("invalid value \"%d\" for flag -grpcping: less than zero\n", grpcInterval)
		os.Exit(2)
	}
	if wsInterval <= 0 {
		fmt.Printf("invalid value \"%d\" for flag -wsping: zero or less\n", wsInterval)
		os.Exit(2)
	}
	zlog := zerolog.New(os.Stderr).Level(zerolog.DebugLevel).With().
		Int("http", httpPort).
		Int("https", httpsPort).
		Int("grpc", grpcPort).
		Int("grpcs", grpcsPort).
		Int("timeout", idleTimeout).
		Int("interval", probeInterval).
		Int("grpcping", grpcInterval).
		Int("wsping", int(wsInterval)).
		//Int("wsmaxsize", int(maxMessageSize)).
		Bool("exec", execFlag).
		Bool("proxy", proxyFlag).
		Bool("nolog", noLogFlag).Logger()

	if metaDataType := getMetaDataType(); metaDataType != "" {
		zlog.Log().Msg("running on AWS")
		store.host.AZ = getAZ(metaDataType)
		if metaDataType == "ec2" {
			runOnEC2 = true
			store.host.InstanceType = getEC2MetaData("instance-type")
		}
	} else {
		zlog.Log().Msg("running on non-AWS")
	}

	hub = newHub()
	pingPeriod = time.Duration(wsInterval) * time.Second // Send pings to peer with this period. Must be less than pongWait.
	pongWait = pingPeriod * 10 / 9                       // Time allowed to read the next pong message from the peer.
	writeWait = 10 * time.Second                         // Time allowed to write a message to the peer.
	maxMessageSize = 1024                                // Maximum message size allowed from peer.
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

func getMetaDataToken() string {
	url := "http://169.254.169.254/latest/api/token"
	client := http.Client{
		Timeout: time.Second,
	}
	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("X-aws-ec2-metadata-token-ttl-seconds", "21600")

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	return string(body)
}

func getFromIMDS(path string) (data string) {
	// can not access imds from docker container when use aws-sdk-go/aws/ec2metadata
	addr := "http://169.254.169.254/latest/meta-data"
	client := http.Client{
		Timeout: time.Second,
	}
	req, err := http.NewRequest("GET", addr+path, nil)
	if err != nil {
		return ""
	}
	if token := getMetaDataToken(); token != "" {
		req.Header.Set("X-aws-ec2-metadata-token", token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	body, err := io.ReadAll(resp.Body)
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

// MetadataTask ... for unmarshal from {ECS_CONTAINER_METADATA_URI_V4}/task
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
	body, err := io.ReadAll(resp.Body)
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
	return "unknown"
}
