FROM gofeatureflag/go-feature-flag:latest

WORKDIR /goff

COPY flags /goff/flags
COPY goff-proxy.yaml /goff/goff-proxy.yaml

ENTRYPOINT ["/go-feature-flag", "-config", "/goff/goff-proxy.yaml"]

