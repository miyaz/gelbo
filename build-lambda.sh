#!/bin/bash

# Build for Lambda
GOOS=linux GOARCH=amd64 go build -o bootstrap .
zip gelbo-lambda.zip bootstrap

echo "Lambda deployment package created: gelbo-lambda.zip"
