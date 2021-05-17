# gelbo
backend app for testing elb

## usage

1. pull and run

   ```
   docker run -d --name gelbo -p 80:9000 public.ecr.aws/h0g2h5b7/gelbo
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

1. ecr login

   ```
   aws ecr-public get-login-password --region us-east-1 | docker login --username AWS --password-stdin public.ecr.aws/h0g2h5b7
   ```

1. build

   ```
   docker build -t gelbo .
   ```

1. tagging

   ```
   docker tag gelbo:latest public.ecr.aws/h0g2h5b7/gelbo:latest
   ```

1. push

   ```
   docker push public.ecr.aws/h0g2h5b7/gelbo:latest
   ```

1. ecr logout

   ```
   docker logout public.ecr.aws
   ```

