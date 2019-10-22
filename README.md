NAV CA Bundle
=============

Fetches production and test certificates and converts them to PEM to the specified dir.

# Usage

```
sudo ./install-certs.sh /etc/pki/ca-trust/source/anchors
sudo update-ca-trust
```

# Mozilla CA certificate bundle

This is a collection of CA certificates included with Mozilla Firefox.
See https://curl.haxx.se/docs/caextract.html.

The file is cached in this repository as `cacert.pem`.

To update the file, run:
```
curl --remote-name --time-cond cacert.pem https://curl.haxx.se/ca/cacert.pem
```

# Additional CA certificates 

Add PEM certificate to additional_ca_certs.cer if the certificate is not available from curl.haxx.se.
