#!/usr/bin/env sh

function dcurl {
    pushd "$1" > /dev/null
    curl -sO "$2"
    popd > /dev/null
}

function download_certs_prod {
    dcurl "$1" "http://crl.adeo.no/crl/eksterne/webproxy.nav.no.crt"

    dcurl "$1" "http://crl.adeo.no/crl/A01PKIROOT2012_NAV%20Root%20CA(1).crt"
    dcurl "$1" "http://crl.adeo.no/crl/A01PKISUB2012_NAV%20Sub%20CA(1).crt"
    dcurl "$1" "http://crl.adeo.no/crl/a01drvw064.adeo.no_NAV%20Issuing%20CA.crt"
    dcurl "$1" "http://crl.adeo.no/crl/A01DRVW006.adeo.no_NAV%20Issuing%20CA%20Intern(1).crt"
    dcurl "$1" "http://crl.adeo.no/crl/A01DRVW269.adeo.no_NAV%20Issuing%20CA%20ekstern(1).crt"
}

function download_certs_dev {
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

function usage {
    echo "Usage: $0 DEST_DIR [prod|dev|all]"
    echo
    echo "Downloads the NAV CA bundle as individual PEM files into a directory."
    echo
    echo "The second parameter specifies whether the bundle should be limited to"
    echo "certificate authorities that only issue certificates for production services."
    echo "Defaults to 'all'."
    exit 1
}

[ $# -lt 1 ] && usage

temp_dir=$(mktemp -d)
dest_dir=$(realpath "$1")
mkdir -p $dest_dir

case $2 in
"prod")
    use_prod=1
    use_dev=0
    ;;
"dev")
    use_prod=0
    use_dev=1
    ;;
"all"|"")
    use_prod=1
    use_dev=1
    ;;
*)
    echo "Syntax error: '$2' is not a valid certificate list specification"
    echo
    usage
esac

[ $use_prod -eq 1 ] && download_certs_prod "$temp_dir"
[ $use_dev -eq 1 ] && download_certs_dev "$temp_dir"

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
