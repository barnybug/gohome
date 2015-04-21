FROM golang:1.4.2-cross
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -yq \
upx \
--no-install-recommends
# goop: managing go deps
RUN go get github.com/nitrous-io/goop
ADD build.sh /
CMD /build.sh
