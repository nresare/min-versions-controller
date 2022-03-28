#!/bin/sh

set -x -e

# Shamelessly stolen from
# https://www.erianna.com/ecdsa-certificate-authorities-and-certificates-with-openssl/

WEBHOOK_CN=podhook.min-versions.svc
WEBHOOK_NS=min-versions
CA_CN=X


#openssl ecparam -genkey -name prime256v1 -out ca.key

#openssl req -x509 -new -key ca.key -nodes -days 3650 -subj "/CN=${CA_CN}" -out ca.crt

#openssl ecparam -genkey -name prime256v1 -out server.key

openssl req -new -key server.key -nodes -subj "/CN=${WEBHOOK_CN}" \
  -out server.csr

tmpfile=$(mktemp /tmp/openssl-config.XXXXXX)
trap "rm -f $tmpfile" 0 2 3 15

cat <<EOF >$tmpfile
[ext]
basicConstraints = CA:FALSE
keyUsage = digitalSignature, keyEncipherment
subjectAltName = DNS:${WEBHOOK_CN}
EOF

openssl x509 -req -SHA256 -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
  -extensions ext -extfile $tmpfile -days 365 -out server.crt

rm -f $tmpfile

kubectl create secret generic webhook-certs \
  --namespace ${WEBHOOK_NS} \
  --from-file=server.key \
  --from-file=server.crt
