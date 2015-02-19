FROM golang:1.3-cross
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -yq \
upx \
--no-install-recommends
# goop: managing go deps
RUN go get github.com/nitrous-io/goop
# goupx: compression of exes
RUN go get github.com/pwaller/goupx
ADD build.sh /
CMD /build.sh
