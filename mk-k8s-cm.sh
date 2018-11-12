#!/bin/bash
#
# This script reads a PEM certificate bundle from STDIN, and generates a
# ConfigMap with both this bundle and a corresponding Java keystore.

pem=`mktemp`
truststore=`mktemp`
outdir=`mktemp -d`

rm $truststore
cat - > $pem

cd $outdir
csplit $pem '/-----BEGIN CERTIFICATE-----/' '{*}'

# mac os x compatibility
if [ $? -ne 0 ]; then
    split -p "-----BEGIN CERTIFICATE-----" $pem
fi

for file in *; do
    echo "--- Processing file $file ---" >&2
    cap=$(openssl x509 -in "$file" -noout -subject)
    if [ $? -eq 0 ]; then
        echo $cap >&2
        keytool -import -noprompt -storepass changeme -alias $file -keystore $truststore -file $file >&2
    fi
done

kubectl \
    --dry-run=true \
    --output=yaml \
    create configmap ca-bundle \
    --from-file=ca-bundle.pem=${pem},ca-bundle.jks=${truststore}

rm -rf $pem
rm -rf $truststore
rm -rf $outdir
