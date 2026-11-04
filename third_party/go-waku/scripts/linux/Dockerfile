# BUILD IMAGE --------------------------------------------------------
FROM ubuntu:22.04
ARG UNAME=jenkins
ARG UID=1001
ARG GID=1001

LABEL maintainer="richard@status.im"
LABEL source="https://github.com/waku-org/go-waku"
LABEL description="go-waku: deb/rpm builder"

RUN export DEBIAN_FRONTEND=noninteractive \
 && apt update -yq \
 && apt install -yq ruby wget git rpm build-essential s3cmd curl

# Installing Golang
RUN GOLANG_SHA256="ef79a11aa095a08772d2a69e4f152f897c4e96ee297b0dc20264b7dec2961abe" \
 && GOLANG_TARBALL="go1.20.11.linux-amd64.tar.gz" \
 && wget -q "https://dl.google.com/go/${GOLANG_TARBALL}" \
 && echo "${GOLANG_SHA256} ${GOLANG_TARBALL}" | sha256sum -c \
 && tar -C /usr/local -xzf "${GOLANG_TARBALL}" \
 && rm "${GOLANG_TARBALL}" \
 && ln -s /usr/local/go/bin/go /usr/local/bin

RUN gem install fpm

# Jenkins user needs a specific UID/GID to work
RUN groupadd -g $GID $UNAME \
 && useradd --create-home -u $UID -g $GID $UNAME
USER $UNAME
ENV HOME="/home/${UNAME}"