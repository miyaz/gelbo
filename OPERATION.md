## usage

1. pull and run

   ```
   docker run -d --name gelbo -p 80:80 -p 443:443 public.ecr.aws/h0g2h5b7/gelbo
   ```

1. watch logs

   ```
   docker logs -f gelbo
   ```

1. stop & remove

   ```
   docker stop gelbo
   docker rm gelbo
   ```

## run locally

1. compile .proto file (only when .proto file is updated)

   ```
   protoc --proto_path ./grpc/proto \
      --go_out=grpc/pb/ --go_opt=paths=source_relative \
      --go-grpc_out=grpc/pb/ --go-grpc_opt=paths=source_relative \
      gelbo.proto
   ```

1. create cert files

   ```
   mkdir -p cert
   openssl req -x509 -nodes -newkey rsa:2048 -days 3650 -keyout cert/server-key.pem -out cert/server-cert.pem -subj "/CN=localhost"
   ```

1. run

   ```
   go run *go
   ```

1. access from local (example commands)

   ```
   curl http://127.0.0.1
   curl -k https://127.0.0.1
   grpcurl -v -proto ./grpc/proto/gelbo.proto -plaintext \
           -d '{"sleep":"1000"}' 127.0.0.1:50051 "elbgrpc.GelboService.Unary"
   grpcurl -v -proto ./grpc/proto/gelbo.proto -insecure \
           -d '{"sleep":"1000","repeqt":"3"}' 127.0.0.1:50052 "elbgrpc.GelboService.BidiStream"
   ```

## container image update

1. ecr login

   ```
   aws ecr-public get-login-password --region us-east-1 | docker login --username AWS --password-stdin public.ecr.aws/h0g2h5b7
   ```

1. build & push

   ```
   docker buildx create --name gelbo --platform amd64,arm64 --bootstrap --use
   docker buildx build --platform linux/amd64,linux/arm64 -t public.ecr.aws/h0g2h5b7/gelbo:latest --push .
   docker buildx rm gelbo
   ```

1. ecr logout

   ```
   docker logout public.ecr.aws
   ```

## copy binary from docker image to host volume

1. build

   ```
   docker build -t gelbo .
   ```

1. copy


   ```
   CONTAINER_ID=`docker create gelbo`
   docker cp $CONTAINER_ID:/app/gelbo .
   docker rm $CONTAINER_ID
   ```
