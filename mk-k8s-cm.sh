#!/bin/bash
#
# This script reads a PEM certificate bundle from STDIN, and generates a
# ConfigMap with both this bundle and a corresponding Java keystore.

pem=`mktemp`
truststore=`mktemp`
out=`mktemp`
outdir=`mktemp -d`

rm $truststore
cat - > $pem

cd $outdir
csplit $pem '/-----BEGIN CERTIFICATE-----/' '{*}' >&2

# mac os x compatibility
if [ $? -ne 0 ]; then
    split -p "-----BEGIN CERTIFICATE-----" $pem >&2
fi

for file in *; do
    echo "--- Processing file $file ---" >&2
    cap=$(openssl x509 -in "$file" -noout -subject)
    if [ $? -eq 0 ]; then
        echo $cap >&2
        keytool -import -noprompt -storepass changeme -alias $file -keystore $truststore -file $file >&2
    fi
done

cat > $out <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: ca-bundle
EOF

echo "binaryData:" >> $out
echo -n "  ca-bundle.jks: " >> $out
base64 < $truststore >> $out

echo "data:" >> $out
echo "  ca-bundle.pem: |" >> $out
sed -E 's/^(.*)$/    \1/g' $pem >> $out

cat $out

rm -rf $pem
rm -rf $truststore
rm -rf $out
rm -rf $outdir
