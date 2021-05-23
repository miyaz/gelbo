#!/bin/sh

## ec2 userdata script

# docker setup
yum install -y docker
usermod -a -G docker ec2-user
systemctl enable docker
systemctl start docker

# gelbo auto start
docker pull public.ecr.aws/h0g2h5b7/gelbo
chmod +x /etc/rc.d/rc.local
echo 'docker pull public.ecr.aws/h0g2h5b7/gelbo' >> /etc/rc.d/rc.local
echo 'docker run -dit --restart always --name gelbo -p 80:80 public.ecr.aws/h0g2h5b7/gelbo' >> /etc/rc.d/rc.local

reboot
