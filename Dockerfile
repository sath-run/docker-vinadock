FROM ubuntu:latest

LABEL buildDate=$buildDate
LABEL "run.sath.author"="Xin Zeng"

ADD ADFRsuite-1.1 /ADFRsuite-1.1
ENV PATH="$PATH:/ADFRsuite-1.1/bin"

WORKDIR /vinadock
COPY bin ./bin
COPY main .
COPY VERSION /
