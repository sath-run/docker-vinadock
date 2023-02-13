FROM ubuntu:latest

LABEL buildDate=$buildDate
LABEL "run.sath.author"="Xin Zeng"

WORKDIR /vinadock
COPY bin ./bin
COPY main .
COPY VERSION /
