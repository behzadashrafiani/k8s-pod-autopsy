FROM gcr.io/distroless/static-debian12
COPY autopsy /autopsy
ENTRYPOINT ["/autopsy"]
