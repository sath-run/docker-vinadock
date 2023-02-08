# exit when any command fails
set -e

push=false;

while getopts 'p' flag; do
  case "${flag}" in
    p) push=true ;;
    *) echo "usage:  "
       exit 1 ;;
  esac
done

env GOOS=linux GOARCH=amd64 go build -o main && \
docker build -t vinadock .
rm ./main

if ! [ "$push" = true ] ; then
    exit 0
fi

VERSION=$( cat VERSION )

docker tag vinadock zengxinzhy/vinadock:latest
docker tag vinadock zengxinzhy/vinadock:$VERSION
docker push zengxinzhy/vinadock:latest
docker push zengxinzhy/vinadock:$VERSION