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

cat > $out_cm_jks <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: ca-bundle-jks
EOF

echo "binaryData:" >> $out_cm_jks
echo -n "  ca-bundle.jks: " >> $out_cm_jks
base64 < $truststore | tr -d '\n' >> $out_cm_jks
echo >> $out_cm_jks

cat > $out_cm_pem <<EOF
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: ca-bundle-pem
EOF

echo "data:" >> $out_cm_pem
echo "  ca-bundle.pem: |" >> $out_cm_pem
sed -E 's/^(.*)$/    \1/g' $pem >> $out_cm_pem

cat $out_cm_jks $out_cm_pem

rm -rf $pem
rm -rf $truststore
rm -rf $out_cm_jks
rm -rf $out_cm_pem
rm -rf $outdir
