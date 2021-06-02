aws ecr-public get-login-password --region us-east-1 |\
  docker login --username AWS --password-stdin public.ecr.aws/h0g2h5b7
docker build -t gelbo . --no-cache
docker tag gelbo:latest public.ecr.aws/h0g2h5b7/gelbo:latest
docker push public.ecr.aws/h0g2h5b7/gelbo:latest
docker logout public.ecr.aws
