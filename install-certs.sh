#!/usr/bin/env sh

function dcurl {
    pushd "$1" > /dev/null
    curl -sO "$2"
    popd > /dev/null
}

function download_certs {
    dcurl "$1" "http://crl.adeo.no/crl/A01PKIROOT2012_NAV%20Root%20CA(1).crt"
    dcurl "$1" "http://crl.adeo.no/crl/A01PKISUB2012_NAV%20Sub%20CA(1).crt"
    dcurl "$1" "http://crl.adeo.no/crl/a01drvw064.adeo.no_NAV%20Issuing%20CA.crt"
    dcurl "$1" "http://crl.adeo.no/crl/A01DRVW006.adeo.no_NAV%20Issuing%20CA%20Intern(1).crt"
    dcurl "$1" "http://crl.adeo.no/crl/A01DRVW269.adeo.no_NAV%20Issuing%20CA%20ekstern(1).crt"

    dcurl "$1" "http://crl.preprod.local/crl/B27pkiroot2012_B27%20Root%20CA(1).crt"
    dcurl "$1" "http://crl.preprod.local/crl/B27pkisub2012_B27%20Sub%20CA(4).crt"
    dcurl "$1" "http://crl.preprod.local/crl/B27DRVW009.preprod.local_B27%20Issuing%20CA(1).crt"
    dcurl "$1" "http://crl.preprod.local/crl/B27DRVW008.preprod.local_B27%20Issuing%20CA%20Intern(1).crt"
    dcurl "$1" "http://crl.preprod.local/crl/B27DRVW056.preprod.local_NAV%20Issuing%20CA%20ekstern.crt"

    dcurl "$1" "http://crl.test.local/crl/D26pkiroot_D26%20Root%20CA(2).crt"
    dcurl "$1" "http://crl.test.local/crl/D26pkisub_D26%20Sub%20CA(2).crt"
    dcurl "$1" "http://crl.test.local/crl/D26DRVW050.test.local_D26%20Issuing%20CA(3).crt"
    dcurl "$1" "http://crl.test.local/crl/D26DRVW051.test.local_D26%20Issuing%20CA%20Intern(2).crt"
}

if [ $# -ne 1 ]
then
    echo "Usage: $0 DEST_DIR"
    exit 1
fi

temp_dir=$(mktemp -d)
dest_dir=$(realpath "$1")

download_certs "$temp_dir"

OIFS="$IFS"
IFS=$'\n'
for file in `find "$temp_dir" -type f -name '*.crt' -o -type f -name '*.cer'`
do
    INFORM=pem
    if ! openssl x509 -in "$file" -noout 2>/dev/null
    then
        if ! openssl x509 -in "$file" -inform der -noout 2>/dev/null
        then
            echo "$file is neither PEM nor DER"
            exit 1
        fi

        INFORM=der
    fi

    new_name=$(openssl x509 -in "$file" -inform $INFORM -noout -subject | sed s/.*CN=// | tr ' ' '_' | tr -d '.')

    echo installing $dest_dir/$new_name.pem from $(basename "$file")
    openssl x509 -in "$file" -inform $INFORM -out "$dest_dir/$new_name.pem"
done
IFS="$OIFS"
