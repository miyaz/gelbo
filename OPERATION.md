## usage

1. pull and run

   ```
   docker run -d --name gelbo -p 80:80 -p 443:443 public.ecr.aws/h0g2h5b7/gelbo
   ```

1. watch logs

   ```
   docker logs -f gelbo
   ```

1. stop

   ```
   docker stop gelbo
   ```

## container image update

1. compile .proto file (only when .proto file is updated)

   ```
   protoc --proto_path ./grpc/proto \
      --go_out=grpc/pb/ --go_opt=paths=source_relative \
      --go-grpc_out=grpc/pb/ --go-grpc_opt=paths=source_relative \
      gelbo.proto
   ```

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
