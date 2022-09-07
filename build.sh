env GOOS=linux GOARCH=amd64 go build -o main && \
docker build -t vinadock .
rm ./main