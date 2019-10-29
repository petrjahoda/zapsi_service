FROM alpine:latest
RUN apk update && apk upgrade && apk add bash && apk add procps && apk add nano
WORKDIR /bin
COPY /linux /bin
ENTRYPOINT zapsi_service_linux
HEALTHCHECK CMD ps axo command | grep dll