#!/bin/bash

base_package=github.com/mitsuse/pushbullet-go
base_path=${GOPATH}/src/${base_package}

package_list=(
    ${base_package}
    ${base_package}/requests
    ${base_package}/responses
)

if [ ! -d ${base_path}/coverprofile ]
then 
    mkdir ${base_path}/coverprofile
else
    rm ${base_path}/coverprofile/*.coverprofile
fi

for package in ${package_list[@]}
do
    cover_name=$(echo ${package} | sed -e "s/\//__/g").coverprofile
    cover_path=${base_path}/coverprofile/${cover_name}
    go test -covermode=count -coverprofile ${cover_path} ${package}
done

cd ${base_path}/coverprofile && gover
