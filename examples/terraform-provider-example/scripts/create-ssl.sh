#!/bin/sh -e

if [ -f server.pem ]; then
    echo certs already exist. Skipping generation
    exit 0
fi

cfssl gencert -initca "$(dirname "$0")/ssl/ca.json" | cfssljson -bare ca
cfssl gencert -ca ca.pem -ca-key ca-key.pem -config "$(dirname "$0")/ssl/cfssl.json" -profile=server "$(dirname "$0")/ssl/server.json" | cfssljson -bare server
