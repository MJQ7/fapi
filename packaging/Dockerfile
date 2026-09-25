# The fapi Docker image. GoReleaser builds the fapi executables first and
# puts each one in a folder named after its platform, such as linux/amd64/,
# in the build context (see dockers_v2 in .goreleaser.yaml).
FROM alpine:3.22

# Set by Docker to the platform being built: linux/amd64.
ARG TARGETPLATFORM

# Run as an ordinary user, not root, with the data folder owned by it.
RUN adduser -D -H fapi && mkdir /data && chown fapi /data
USER fapi

COPY $TARGETPLATFORM/fapi /usr/bin/fapi
COPY packaging/docker-config.json /etc/fapi/config.json

# Data (endpoints and logs) in the /data volume; settings from the file
# above, which listens on all addresses so Docker can publish the ports.
# Use your own settings file by mounting it and changing FAPI_CONFIG.
ENV FAPI_DATA_DIR=/data
ENV FAPI_CONFIG=/etc/fapi/config.json
VOLUME /data

# The admin port. Mock ports are published with `docker run -p` as needed.
EXPOSE 3100

ENTRYPOINT ["/usr/bin/fapi"]
CMD ["serve"]
