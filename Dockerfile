FROM alpine:3.16
WORKDIR /vinadock
COPY bin ./bin
COPY main .
ENTRYPOINT [ "./main" ]