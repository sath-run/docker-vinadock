env GOOS=linux GOARCH=amd64 go build -o main.o && \
docker build -t vinadock .
rm ./main.o