FROM centos:7

RUN yum install -y smartmontools lsscsi

COPY ./_build/local-disk-manager /local-disk-manager

ENTRYPOINT [ "/local-disk-manager" ]
