# gelbo
backend app for testing elb

1. pull and run

```
docker run -dit --rm --name gelbo -p 9000:9000 public.ecr.aws/h0g2h5b7/gelbo
```

2. watch logs

```
docker logs -f gelbo
```

3. stop

```
docker stop gelbo
```
