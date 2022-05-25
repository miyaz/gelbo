#/bin/bash

for feature in `ls ./proto | sed 's/\.proto//g'`
do
  echo $feature
  mkdir -vp pb/$feature
  protoc --proto_path ./proto --go_out=plugins=grpc:./pb/$feature $feature.proto
done
