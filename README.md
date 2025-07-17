Certificator
============

Certificator is a daemon that maintains CA bundles as Kubernetes configmaps.

At regular intervals, Certificator will load PEM or DER data from specified URLs and directories.
The certificate data will be validated for correctness, and added to a cache.
The cached certificates are then persisted into all eligible Kubernetes namespaces.

## Configuration

| Environment variable                  | Type                           | Default        |
|---------------------------------------|--------------------------------|----------------|
| CERTIFICATOR_CA_URLS                  | Comma-separated list of String |                |
| CERTIFICATOR_CA_DIRECTORIES           | Comma-separated list of String |                |
| CERTIFICATOR_DOWNLOAD_TIMEOUT         | Duration                       | 5s             |
| CERTIFICATOR_DOWNLOAD_INTERVAL        | Duration                       | 24h            |
| CERTIFICATOR_DOWNLOAD_RETRY_INTERVAL  | Duration                       | 10m            |
| CERTIFICATOR_APPLY_BACKOFF            | Duration                       | 5m             |
| CERTIFICATOR_APPLY_TIMEOUT            | Duration                       | 10s            |
| CERTIFICATOR_JKS_PASSWORD             | String                         | changeme       |
| CERTIFICATOR_LOG_FORMAT               | LogFormat                      | text           |
| CERTIFICATOR_LOG_LEVEL                | LogLevel                       | debug          |
| CERTIFICATOR_METRICS_ADDRESS          | String                         | 127.0.0.1:8080 |
| CERTIFICATOR_NAMESPACE_LABEL_SELECTOR | String                         | team           |

It is recommended to add the [Mozilla certificate store](https://curl.se/ca/cacert.pem)
as one of the URLs. See [CA Extract](https://curl.se/docs/caextract.html) for details.

Run `certificator --help` for more information.

## Why

Certain legacy services at NAV use certificates signed by an internal certificate authority.
These CA certificates are not included in any Linux distribution. Thus, when building a Docker image,
the author must include these certificates manually in order to speak securely to said services. 
The role of Certificator is to remove this inconvenience.

By bundling upstream CA certificates together with these internal NAV certificates, Certificator
creates a new bundle that can be mounted into the pods directly.
[Naiserator mounts these files automatically](https://github.com/nais/naiserator/blob/master/pkg/resourcecreator/certificateauthority/certificateauthority.go).

Furthermore, Certificator exposes the certificate bundles both in PEM format,
and also Java Keystore format, suitable for Java applications.

## Verifying the certificator images and their contents

The images are signed "keylessly" using [Sigstore cosign](https://github.com/sigstore/cosign).
To verify their authenticity run
```
cosign verify \
--certificate-identity "https://github.com/nais/certificator/.github/workflows/release.yml@refs/heads/master" \
--certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
europe-north1-docker.pkg.dev/nais-io/nais/images/certificator@sha256:<shasum>
```

The images are also attested with SBOMs in the [CycloneDX](https://cyclonedx.org/) format.
You can verify these by running

```
cosign verify-attestation --type cyclonedx  \
--certificate-identity "https://github.com/nais/certificator/.github/workflows/build_and_push_image.yaml@refs/heads/master" \
--certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
europe-north1-docker.pkg.dev/nais-io/nais/images/certificator@sha256:<shasum>
```
