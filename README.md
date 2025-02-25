# ELB Verification Tool “gelbo”

## What is “gelbo”?

* A web server
  * Implemented using Go language’s net/http package.
  * Designed to run on the ELB target instances (mainly ALB/CLB).
  * Displays various controls and information depending on the specified path and query string.
* A Docker container
  * You can use it if you have a Docker environment (EC2, ECS, etc.).
  * The container image is published in the Public ECR [https://gallery.ecr.aws/h0g2h5b7/gelbo].
  * The image size (after compression) is only 7MB.
* When to use (Examples)
  * Maintain a specified CPU/memory usage rate to verify an Auto Scaling’s scaling policy.
    * Resource control (cpu/memory) function
  * Check the behavior of application-based sticky sessions.
    * Response header control (addheader/delheader) function
  * Check the routing bias.
    * Request information confirmation or statistics confirmation function
  * Delay response only on a specified target (or ELB node) in AZ-1a.
    * If condition specification function AND Response control (sleep/size/status/chunk) function

## Installation methods

* Run the below comment in a Docker environment.
  * If you specify '--restart=always', it automatically starts up when the process goes down.

```
docker run -d --restart=always -p 80:80 -p 443:443 --name gelbo public.ecr.aws/h0g2h5b7/gelbo
```

* Other execution examples:

```
# Stop the container
docker stop gelbo
# Delete the container
docker rm gelbo
# Update the container image
docker pull public.ecr.aws/h0g2h5b7/gelbo
```

### Example of using gelbo on Amazon Linux 2 or 2023

* Run docker run after installing Docker.
* The following is an example of describing the necessary preprocessing in the userdata (script) and running ec2 run-instances.
  * Note: The hop limit is changed in the metadata option, specifying “HttpPutResponseHopLimit=2” to enable retrieving the session token from the Docker container in the IMDSv2 environment.

```
# Create a script file for userdata
cat << EOS > userscript.txt
#!/bin/bash
RES=1
# Retry the docker installation process becuase it can fail sometimes
for cnt in { 1..12 }
do
  if [ \$RES -ne 0 ]; then
    yum install -y docker | grep -q "Nothing to do"
    RES=\$?
    sleep 5
  fi
done
systemctl start docker.service
systemctl enable docker.service
usermod -a -G docker ec2-user

docker run -d --restart=always -p 80:80 -p 443:443 --name gelbo public.ecr.aws/h0g2h5b7/gelbo
EOS

# Specify the above sript file and launch the EC2 instance
aws ec2 run-instances --region ap-northeast-1 \
    --user-data file://userscript.txt \
    --image-id ami-012345678901234567  --instance-type t3.micro \
    --key-name ${KEYPAIR_NAME} --security-group-ids ${SG_ID} \
    --metadata-options "HttpEndpoint=enabled,HttpTokens=required,HttpPutResponseHopLimit=2"
```

* To run it directly on an EC2 instance (Linux) instead of Docker, execute the below after logging into the EC2 instance.
* Extract and run gelbo binary from the Docker image.

```
# Extract the gelbo binary from the Docker image
IMAGE_NAME=public.ecr.aws/h0g2h5b7/gelbo
FILEPATH=/app/gelbo
CONTAINER_ID=`docker create $IMAGE_NAME`
docker cp $CONTAINER_ID:$FILEPATH .
docker rm $CONTAINER_ID

# Disable Docker startup if necessary
sudo systemctl stop docker.service
sudo systemctl disable docker.service 

# Start gelbo
nohup sudo ./gelbo >> gelbo.log 2>&1 &
```

## Command-line Options

Currently, gelbo supports the following options:

* -http {http port number}
  * The port to use for the HTTP protocol (default: 80)
* -https {https port number}
  * The port to use for the HTTPS protocol (default: 443)
* -timeout {timeout seconds}
  * The keep-alive timeout value. (default: 65).
  * TCP connection will be disconnected after the specified time.
  * TCP connection will be kept alive if the value is 0 (gelbo will not disconnect).
* -wsping {WebSocket ping frame transmission interval (seconds))
  * The interval to send Ping frames to the client on the WebSocket connection (default: 30).
  * Specify a value greater than 0.
* -exec
  * Enables the arbitrary command execution feature.
* -proxy
  * Specify to support Proxy Protocol v1/v2 communication (retrieves the client IP address from the Proxy Protocol header).
  * (Background) Even when enabled, it can also handle non-Proxy Protocol communication, but there are rare cases where communication delays (sudden delays of tens of seconds) occur when Proxy Protocol is enabled. Therefore, it is an option and is disabled by default. Enable it when you need to use Proxy Protocol.
* -nolog
  * Specify to not output logs.

* Specify these options at the end of the docker run command as below:

```
docker run -d --restart=always -p 80:80 -p 443:443 --name gelbo public.ecr.aws/h0g2h5b7/gelbo -timeout 65
```

# Functions

## Request Information Confirmation

* Displays details of the requests the target receives (refer to the below parts in blue).

```
% curl "gelbo-xxxxxxxxx.ap-northeast-1.elb.amazonaws.com/?key1=value1&key2=value2"
{
  "host": {
    "name": "ip-172-31-27-115.ap-northeast-1.compute.internal",
    "ip": "172.31.27.115",
    "az": "ap-northeast-1d",
    "type": "t3.small"
  },
  "resource": {
    (snip)
  },
  "request": {
    "protocol": "http",
    "method": "GET",
    "path": "/",
    "querystring": "key1=value1&key2=value2",
    "header": {
      "Accept": "*/*",
      "Host": "gelbo-xxxxxxxxx.ap-northeast-1.elb.amazonaws.com",
      "User-Agent": "curl/7.64.1",
      "X-Amzn-Trace-Id": "Root=1-60d04f82-14983854025af6ac3e1ff49e",
      "X-Forwarded-For": "203.0.113.145, 172.31.39.211",
      "X-Forwarded-Port": "80",
      "X-Forwarded-Proto": "http"
    },
    "clientip": "203.0.113.145",
    "proxy1ip": "172.31.39.211",
    "lasthopip": "172.31.23.250"
    "targetip": "172.17.0.2"
  },
  "direction": {
    (snip)
  }
}
```

### Description

* Displays the protocol, method, headers, path, and query string of the request.
  * You can find the information in the request field:  protocol, method, path, querystring, header.
  * The protocol values and descriptions:
    * http - HTTP
    * https - HTTP over TLS
    * h2c - HTTP/2
    * h2 - HTTP/2 over TLS
* Displays various IP addresses:
  * clientip: the IP address of the request origin.
    * It retrieves the client’s IP address from the X-Forwarded-For header. If the X-Forwarded-For header is not available, it retrieves the IP address of the connection origin. 
  * proxy1ip ~ proxy3ip: the IP addresses of the proxy servers between the clientip and lasthopip (displays up to three IP addresses).
    * Displayed in a multi-tier configuration, for example ALB. 
    * In the following configuration, the IP address of each node is the value in the corresponding field:
      * Client (clientip) → ALB1 (proxy1ip) → nginx (proxy2ip) → ALB2 (lasthopip) → gelbo (targetip)
  * lasthopip: the IP address of the Last Hop proxy server.
    * When using ALB/CLB, if there is X-Forwarded-For header, it retrieves the IP address of the connection origin (the private IP address of the ELB node).
  * targetip: the IP address of the target.
    * When using Docker, it retrieves the IP address of the container. Refer to host.ip for the IP address of the EC2 instance host.
* ```
Displays the contents of the certificate in .request.mtlscert field, If a header for mTLS (“X-Amzn-Mtls-Clientcert” or “X-Amzn-Mtls-Clientcert-Leaf”) is included.

* command e.x.)  curl --key client_key.pem --cert client_cert.pem "https://{gelbo domain}/" | jq -r .request.mtlscert
* output in a format similar to result of `openSSL x509 -text -noout -in {cert_file}`
* Displays the host information:
  * Displays the name and IP address of the host in host.name and host.ip. 
  * Displays host.az (AvailabilityZone) and host.type (InstanceType) if the information can be retrieved from IMDS (169.254.169.254),
    * In case of running gelbo as a Docker container and IMDSv1 is not available, you can set the HopLimit to 2 to retrieve the information from IMDSv2:
    * `aws ec2 modify-instance-metadata-options --instance-id {instance ID} --http-put-response-hop-limit 2`

## Response Control (sleep/size/status/chunk)

* Responds according to sleep (response delay time), size (response size), and status (status code), chunk (chunked transfer) specified in the query string. 

```
% curl "gelbo-xxxxxxxxx.ap-northeast-1.elb.amazonaws.com/?size=1000-2000&sleep=500-4000&status=60"
{
    (snip)
  "direction": {
    "input": {
      "sleep": "500-4000",
      "size": "1000-2000",
      "status": "60"
    },
    "result": {
      "sleep": "2158",
      "size": "1993",
      "status": "invalid"
    }
  }
}


WW3JmB8siYgY5Ijbh4RyTlQU2UfTGvoPfu1fHDeDJo8TbxLP4NqHvRyS966Q2BI1iDaZlkpegDR4mOnKhx8wvpgNTUcFkrLtwbf
L4jGiMXWdk36Ll9BtgOe29YlL9Ktwciqv2SpLutyvcpjdPujhyoEiBiUfXrRspItbx99oQUBEb3yd5BOauhCg1YdbdaFVpcVUTs
(〜Add random characters to the end of the response to reach the specified size.〜)
```

### Description

* sleep=minimum[-maximum]
  * Responds after the specified millisecond duration sleep.
  * Uses a random value within the specified range.
* size=minimum[-maximum]
  * Responds with response size in the specified number of bytes. 
  * Adds random characters to the end of the JSON response to reach the specified size. 
  * Uses a random value within the specified range.
* status=specified status code within the range of 100 to 999
  * Responds with the specified status code. 
* chunk  Responds data chunked (Transfer-Encoding: chunked)
  * Arbitrary value can be specified in this field (not affect operation)
  * Only supports HTTP/1.1
* direction.result
  * Responds "invalid" if you specify an unexpected value (not a number of hyphen).
    * In the example above, 60 is not a valid value for the status, so the response is "invalid".
  * Displays a randomly determined value if a range is specified. 

## Resource Control (cpu/memory)

* Maintains the resource (cpu/memory) usage rate at the value specified in the query string. 

```
% curl "gelbo-xxxxxxxxx.ap-northeast-1.elb.amazonaws.com/?cpu=70&memory=50"
{
    (snip)
  "resource": {
    "cpu": {
      "target": 70,
      "current": 71
    },
    "memory": {
      "target": 50,
      "current": 48.384333048418725
    }
  },
    (snip)
```

### Description

* cpu=0~100
  * Maintains the specified usage rate. 
  * resource.cpu.target is the value specified in the query string.
  * resource.cpu.current is the current usage rate. 
* memory=0~100
  * Maintains the specified usage rate “as much as possible". 
  * resource.memory.target is the value specified in the query string.
  * resource.memory.current is the current usage rate. 
* Things to keep in mind
  * If you set memory=100, it will crash (memory allocation failure).
  * Resource Control may not work if you use ECS (Fargate), because the resource usage in the container and the usage rate observed on the ECS side may differ depending on the resource.
    * For example, cpu=20 may be 100% on the CloudWatch metrics. Adjust the value according to the actual usage to get the desired performance.

## Response Header Control  (addheader/delheader)

* You can add/remove response headers. 

```
% curl -vs "gelbo-xxxxxxxxx.ap-northeast-1.elb.amazonaws.com/?addheader=`echo -n 'Set-Cookie: sticky=1' | jq -s -R -r @uri`" > /dev/null

> GET /?addheader=Set-Cookie%3A%20sticky%3D1 HTTP/1.1
> Host: gelbo-xxxxxxxxx.ap-northeast-1.elb.amazonaws.com
> User-Agent: curl/7.64.1
> Accept: */*
>
< HTTP/1.1 200 OK
< Date: Fri, 25 Jun 2021 09:44:18 GMT
< Content-Type: application/json
< Content-Length: 987
< Connection: keep-alive
< Set-Cookie: sticky=1

% curl -vs "gelbo-xxxxxxxxx.ap-northeast-1.elb.amazonaws.com/?delheader=`echo -n 'Set-Cookie' | jq -s -R -r @uri`" > /dev/null

> GET /?addheader=Set-Cookie%3A%20sticky%3D1 HTTP/1.1
> Host: gelbo-xxxxxxxxx.ap-northeast-1.elb.amazonaws.com
> User-Agent: curl/7.64.1
> Accept: */*
>
< HTTP/1.1 200 OK
< Date: Fri, 25 Jun 2021 09:44:18 GMT
< Content-Type: application/json
< Content-Length: 987
< Connection: keep-alive
```

### Description

* addheader=: the header to include in the response header after the addheader= parameter.
  * Included in all subsequent responses until it is removed by delheader.
  * Use “jq -s -R -r @uri” because URL encoding is required.
    * Because some browsers, such as Chrome, do not encode semicolons, you can use curl + jq as shown in the example above as a safe approach. 
* delheader=: the header to delete.
* You can use this to verify the behavior of application-based sticky sessions,etc.

## “if condition” Specification

* You can use an if statement to specify the target out of multiple targets under ELB to execute size/cpu,etc. 

```
% curl "gelbo-xxxxxxxxx.ap-northeast-1.elb.amazonaws.com/?sleep=100-3000&ifaz=ap-northeast-1a&ifaz=ap-northeast-1c"
{
  "host": {
    "name": "ip-172-31-27-115.ap-northeast-1.compute.internal",
    "ip": "172.31.27.115",
    "az": "ap-northeast-1a",
    "type": "t3.small"
  },
  (snip)

  "direction": {
    "input": {
      "sleep": "100-3000",
      "ifaz": "ap-northeast-1a or ap-northeast-1c"
    },
    "result": {
      "sleep": "1627",
      "ifaz": "matched"
    }
  }
}
```

### Description

* Available if conditions:
  * ifclientip, ifproxy1ip, ifproxy2ip, ifproxy3ip, iflasthopip, iftargetip, ifhostip, ifhost, ifaz, iftype
  * You can specify IPv6 addresses for the IP addresses (exclude the brackets [ ] in this case). 
* You can specify multiple, different if conditions. In this case, it's AND evaluation.
* You can specify multiple, same if conditions. In this case, it's OR evaluation.
* Executes the specified processing when all conditions are met. 
* In the example above, if the target is running in ap-northeast-1a or ap-northeast-1c, it will sleep for 100–3000 milliseconds (randomly determined). 
* You can find the result of the condition evaluation (matched or unmatched) under direction.result.

## Arbitrary Command Execution

* You can execute arbitrary commands from the container. 

```
% curl "127.0.0.1/exec/?cmd=`echo -n 'apk update' | jq -s -R -r @uri`"
〜
OK: 13888 distinct packages available

% curl "127.0.0.1/exec/?cmd=`echo -n 'apk add curl' | jq -s -R -r @uri`"
〜
(3/4) Installing libcurl (7.77.0-r0)
(4/4) Installing curl (7.77.0-r0)
Executing busybox-1.32.1-r6.trigger
OK: 8 MiB in 19 packages

% curl "127.0.0.1/exec/?cmd=`echo -n 'curl 169.254.169.254/latest/meta-data/hostname' | jq -s -R -r @uri`"
ip-172-31-42-159.ap-northeast-1.compute.internal
```

### Description

* /exec/?cmd={arbitrary command} allows you to execute arbitrary commands on the container.
  * In the example above, it installs curl and then uses curl to make a request to the IMDS to retrieve the hostname.
  * Because URL encoding is required with curl, it uses ‘jq -s -R -r @uri’.
  * Because it's based on Alpine, there are few initial installed commands available. Refer to the example and install the necessary commands as needed.
* You can use this to check the environment variables passed to the container in ECS, or to check the connectivity from the container to the outside.
* From a security perspective, be aware of the following when using this function:
  * This function is disabled by default. Start it up by specifying the -exec option only when you need to.
  * Restrict access origins and minimize permissions to retrieve IAM credentials from IMDS to prevent facilitating attacks or leaking credentials.
  * Commands containing the string “credentials/” will not be executed.

## Stopping the container

* You can stop the container. 

```
% curl "gelbo-xxxxxxxxx.ap-northeast-1.elb.amazonaws.com/stop/"
<html>
<head><title>502 Bad Gateway</title></head>
<body>
<center><h1>502 Bad Gateway</h1></center>
</body>
</html>
```

### Description

* Stop the container by using /stop/.
* Restart automatically if you specify '--restart=always- when running docker run.
* You can use this to intentionally bring down the process at any time, such as when verifying the behavior of Auto Scaling self-healing.

## Statistics Confirmation

* Displays the cumulative request count and bytes sent/received, the current TCP connection count and active requests. 

```
% curl "gelbo-xxxxxxxxx.ap-northeast-1.elb.amazonaws.com/monitor/"

{
  "created_at": "2021-06-28T07:49:03Z",
  "updated_at": "2021-06-28T09:39:13Z",
  "request_count": 313,
  "sent_bytes": "1.3 GB",
  "received_bytes": "323.1 MB",
  "cpu": 0.0,
  "memory": 17.5,
  "active_conns": 0,
  "total_conns": 2,
  "elbs": {
    "172.31.24.142": {
      "created_at": "2021-06-28T07:49:20Z",
      "updated_at": "2021-06-28T09:39:07Z",
      "request_count": 156,
      "sent_bytes": "536.8 MB",
      "received_bytes": "160.4 MB",
      "cpu": 0.0,
      "memory": 0.0,
      "active_conns": 0,
      "total_conns": 0
    },
    "172.31.43.209": {
      "created_at": "2021-06-28T07:49:20Z",
      "updated_at": "2021-06-28T09:39:07Z",
      "request_count": 157,
      "sent_bytes": "787.2 MB",
      "received_bytes": "162.7 MB",
      "cpu": 0.0,
      "memory": 0.0,
      "active_conns": 0,
      "total_conns": 2
    }
  }
}
```

### Description

* /monitor/ displays the following information for the target (or for each ELB node it went through):
  * request_count - the number of requests
  * sent_bytes - the response size (excluding the header)
  * received_bytes - the request size (excluding the header)
  * total_conns - the number of TCP connections (e.g. keep-alive)
  * active_conns - the number of active requests
* The IP addresses under "elbs" represent the ELB nodes. 
* Displayed in a human-readable format (e.g. "787.2 MB") by default, but you can specify "?raw" in the query string to get the raw data.
* You can use this to check the distribution (bias) of the requests.

## Logging

* Outputs the access logs in JSON format to standard output (example output below): 

```
{"reqtime":"2022-08-24T06:47:01.413296059Z","proto":"http","method":"POST","path":"/","qstr":"size=100-10000000&sleep=3-10000","clientip":"203.0.113.146","srcip":"172.31.37.32","srcport":32914,"reqsize":12386935,"size":1228766,"status":200,"time":"2022-08-24T06:47:08.593322118Z","duration":7180,"reuse":0}
{"reqtime":"2022-08-24T06:47:09.268519354Z","proto":"http","method":"GET","path":"/","qstr":"size=100-100000&sleep=3-10000","clientip":"203.0.113.146","srcip":"172.31.37.32","srcport":60888,"reqsize":0,"size":90553,"status":200,"time":"2022-08-24T06:47:14.118517026Z","duration":4849,"reuse":0}
{"reqtime":"2022-08-24T06:47:31.002643657Z","proto":"http","method":"GET","path":"/albhealth","qstr":"","clientip":"172.31.25.244","srcip":"172.31.25.244","srcport":10088,"reqsize":0,"size":676,"status":200,"time":"2022-08-24T06:47:31.002735857Z","duration":0,"reuse":0}
```

### Description

* The logged fields include:
  * reqtime - request received time
  * proto - protocol, same as the "protocol" in 'Request Information Confirmation' section
  * method - method
  * path - path
  * qstr - query string
  * clientip - client IP address (retrieved from X-Forwarded-For header if present)
  * srcip - source IP address
  * srcport - source port
  * reqsize - request size (excluding the header)
  * size - response size (excluding the header)
  * status - status code
  * time - response time
  * duration - time elapsed until response (in millisecond)
  * reuse - ・・・number of times the same connection was reused
* Refer to the logs using ‘docker logs gelbo -f -n10’, etc. when using Docker containers.
* Use the -nolog option to disable log output (to avoid heavy I/O load. etc.).

## Environment Variable (Value) Confirmation

* Displays the values of the environment variables. 

```
% curl "gelbo-xxxxxxxxx.ap-northeast-1.elb.amazonaws.com/env/?key=ECS_CONTAINER_METADATA_URI_V4&key=HOME"
http://169.254.170.2/v4/2612bc9219074b7ba718fbac6bd2bb98-3303031112
/root
```

### Description

* /env/?key={environment variable name} displays the value of the specified environment variable.
* You can specify multiple key parameters (as shown in the example above).
* You can specify different environment variables in the ECS task checking them switching during a blue/green deployment, etc.

## WebSocket

* This is a chat function using WebSocket.

![](https://raw.githubusercontent.com/miyaz/gelbo/images/readme/websocket.png)

### Description

* Accessing http[s]://{domain}/chat/ downloads the HTML of the chat screen, and the JavaScript processing will start the WebSocket communication.
* The WebSocket connection destination will automatically become http[s]://{domain}/ws/.
* Display items:
  * Current Online - the current number of connections
  * Connected Server - the IP address of the WebSocket server
  * ClientId - a string that identifies the connected client
    * It is a concatenated string of [X-Forwarded-For,]RemoteAddr,LocalAddr separated by commas.
    * For easy identification, each ClientId is displayed in a different color.
    * When the WebSocket communication ends, the ClienId is struck through.
* You can specify the interval (in seconds) for sending Ping frames with the -wsping option.
  * Specify a value smaller than the ELB idle timeout. If you specify a larger value, the connection is terminated by the ELB if there is no message sent or received during the idle timeout period. 
* Button actions:
  * Connect - starts the WebSocket connection when not connected.
  * Disconnect - disconnects the existing WebSocket connection.
  * Post - sends the text entered in the left text field. This is distributed to other connected clients (chat function).
  * Echo - sends the text entered in the right text field. The same message is returned only to the client (echo function).
* Logged fields:
  * conntime - connection start time
  * proto - protocol, same as the "protocol" in 'Request Information Confirmation' section
  * clientip - client IP address (retrieved from X-Forwarded-For header if present)
  * srcip - source IP address
  * srcport - source port
  * readtime - time of server receiving message from client (not recorded when sending messages)
  * writetime - time of server sending message to client (not recorded when receiving messages)
  * msgsize - message size (in bytes)
  * error - error message (recorded only when an error occurs)

## Other Functions

*  Supports Proxy Protocol v1/v2 (specify -proxy when starting up).
  * v1 can be enabled on [CLB](https://docs.aws.amazon.com/elasticloadbalancing/latest/classic/enable-proxy-protocol.html) , and v2 can be enabled on [NLB](https://docs.aws.amazon.com/elasticloadbalancing/latest/network/edit-target-group-attributes.html#proxy-protocol). 
* Outputs the specified text in gelbo’s standard output and standard error output if you specify stdout and stderr in the query string.
  * Example: /?stdout=foo&stderr=bar
  * Consider using the -nolog option as needed.
* Docker images are compatible with amd64 / arm64 architectures.

