#!/bin/bash

profile=coverprofile/gover.coverprofile
service=wercker.com

goveralls -coverprofile ${profile} -service ${service} -repotoken ${COVERALLS_TOKEN}
