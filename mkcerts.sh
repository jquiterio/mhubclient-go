#!/bin/bash
# call
# ./mkcerts.sh

mkdir certs
rm -rf certs/*

openssl req -new -nodes -x509 \
  -out certs/client.pem \
  -keyout certs/client.key \
  -days 365 \
  -subj "/C=DE/ST=NRW/L=Earth/O=Oops/OU=IT/CN=random.com/emailAddress=me@random.com"

