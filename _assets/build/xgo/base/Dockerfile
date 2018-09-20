FROM karalabe/xgo-1.10.x

# Inject the old Go package downloader and tool-chain bootstrapper
ADD bootstrap.sh /bootstrap.sh
ENV BOOTSTRAP /bootstrap.sh
RUN chmod +x $BOOTSTRAP

# Inject the C dependency cross compiler
ADD build_deps.sh /build_deps.sh
RUN chmod +x $BUILD_DEPS

# Inject the container entry point, the build script (patched for Status bindings conditional builds of C code)
ADD build.sh /build.sh
ENV BUILD /build.sh
RUN chmod +x $BUILD

ENTRYPOINT ["/build.sh"]
