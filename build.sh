#!/bin/bash

set -e

rm -fr test/.terraform
rm -fr test/.terraform.lock.hcl

if [[ $(uname) == "Darwin" ]]; then
    GOOS="darwin"
elif [[ $(uname) == "Linux" ]]; then
    GOOS="linux"
fi

if [[ $(uname -m) == "x86_64" ]]; then
    GOARCH="amd64"
else
    echo "We don't have this arch!."
fi

FILE_NAME="terraform-provider-confluent-kafka_v0.1.0"
DIR=${HOME}/.terraform.d/plugins/registry.terraform.io/hashicorp/confluent-kafka/0.1.0/${GOOS}_${GOARCH}
mkdir -p ${DIR}

go build -o ${FILE_NAME} main.go

mv ${FILE_NAME} $DIR

(cd test && terraform init -get=false)