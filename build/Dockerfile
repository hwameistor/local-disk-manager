FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/local-disk-manager \
    USER_UID=1001 \
    USER_NAME=local-disk-manager

# install operator binary
COPY build/_output/bin/local-disk-manager ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
