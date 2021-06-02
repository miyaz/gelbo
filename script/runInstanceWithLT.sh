INS_ID=`aws ec2 run-instances --launch-template LaunchTemplateName=gelbo |\
jq -r '.Instances[].InstanceId'`
aws ec2 describe-instances --instance-ids $INS_ID |\
jq -r '.Reservations[].Instances[].PublicIpAddress'
