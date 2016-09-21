FROM karalabe/xgo-latest

# Inject the container entry point, the build script (patched for Status bindings conditional builds of C code)
ADD build.sh /build.sh
ENV BUILD /build.sh
RUN chmod +x $BUILD

ENTRYPOINT ["/build.sh"]