#!/bin/bash

## ec2 userdata script

# docker setup
yum install -y docker
usermod -a -G docker ec2-user
systemctl enable docker
systemctl start docker

# gelbo start
docker run -dit --restart always --name gelbo -p 80:80 public.ecr.aws/h0g2h5b7/gelbo
