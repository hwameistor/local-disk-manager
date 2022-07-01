FROM  ghcr.io/hwameistor/local-disk-manager:v0.0.2-amd64

COPY /storcli64 /usr/local/bin/storcli

COPY ./_build/local-disk-manager /local-disk-manager

ENTRYPOINT [ "/local-disk-manager" ]
