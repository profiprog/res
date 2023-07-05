FROM scratch
COPY res /
ENTRYPOINT ["/res"]
WORKDIR /work
