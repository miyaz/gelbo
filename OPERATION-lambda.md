# Lambda Target Support

gelbo can also run as an ALB Lambda target.

## Build

```bash
./build-lambda.sh
```

This creates `gelbo-lambda.zip`.

## Deployment

1. Set the Lambda function name as an environment variable
```bash
export FUNCTION_NAME=gelbo-lambda
```

2. Create a Lambda execution role
```bash
aws iam create-role \
  --role-name lambda-${FUNCTION_NAME}-execution-role \
  --assume-role-policy-document '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"lambda.amazonaws.com"},"Action":"sts:AssumeRole"}]}'

aws iam attach-role-policy \
  --role-name lambda-${FUNCTION_NAME}-execution-role \
  --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole
```

3. Create the Lambda function
```bash
aws lambda create-function \
  --function-name ${FUNCTION_NAME} \
  --runtime provided.al2023 \
  --timeout 65 \
  --role arn:aws:iam::$(aws sts get-caller-identity --query Account --output text):role/lambda-${FUNCTION_NAME}-execution-role \
  --handler bootstrap \
  --zip-file fileb://gelbo-lambda.zip
```

4. Create a target group
```bash
TARGET_GROUP_ARN=$(aws elbv2 create-target-group \
  --name ${FUNCTION_NAME}-tg \
  --target-type lambda \
  --query 'TargetGroups[0].TargetGroupArn' --output text)
```

5. Grant Elastic Load Balancing permission to invoke the Lambda function
```bash
aws lambda add-permission \
  --function-name ${FUNCTION_NAME} \
  --statement-id elb1 \
  --principal elasticloadbalancing.amazonaws.com \
  --action lambda:InvokeFunction \
  --source-arn ${TARGET_GROUP_ARN}
```

6. Register the Lambda function with the target group
```bash
FUNCTION_ARN=$(aws lambda get-function \
  --function-name ${FUNCTION_NAME} \
  --query 'Configuration.FunctionArn' --output text)

aws elbv2 register-targets \
  --target-group-arn ${TARGET_GROUP_ARN} \
  --targets Id=${FUNCTION_ARN}
```

## Updating the Function

To update the Lambda function after modifying the source code:

```bash
./build-lambda.sh

aws lambda update-function-code \
  --function-name ${FUNCTION_NAME} \
  --zip-file fileb://gelbo-lambda.zip
```

## Limitations

The following features are not available in the Lambda environment:
- WebSocket (/ws/, /chat/)
- gRPC
- Resource control (cpu/memory)
- Arbitrary command execution (/exec/)
- Container stop (/stop/)

## Available Features

- Request information display
- Response control (sleep/size/status)
- Response header control (addheader/delheader)
- Conditional execution (if condition)
- Environment variable display (/env/)
- Statistics display (/monitor/)
