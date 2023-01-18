FROM golang:1.18 AS builder

LABEL maintainer="darmiel <hi@d2a.io>"

WORKDIR /usr/src/app
SHELL ["/bin/bash", "-o", "pipefail", "-c"]

# Install dependencies
# Thanks to @montanaflynn
# https://github.com/montanaflynn/golang-docker-cache
COPY go.mod go.sum ./
RUN go mod graph | awk '{if ($1 !~ "@") print $2}' | xargs go get

# Copy remaining source
COPY . .


# Build from sources
RUN GOOS=linux \
    GOARCH=amd64 \
    CGO_ENABLED=0 \
    go build -o bot .

###

FROM alpine:3.15
COPY --from=builder /usr/src/app/bot .
COPY --from=builder /usr/src/app/discord.toml .

# Install python/pip
ENV PYTHONUNBUFFERED=1
RUN apk add --update --no-cache python3 && ln -sf python3 /usr/bin/python
RUN python3 -m ensurepip
RUN pip3 install --no-cache --upgrade pip setuptools

# Install Git
RUN apk add git

# Clone IRDB
RUN git clone https://github.com/logickworkshop/Flipper-IRDB.git Flipper-IRDB-official
RUN git clone https://github.com/darmiel/flipper-scripts.git
RUN git clone https://github.com/darmiel/fff-ir-lint.git

ENTRYPOINT [ "/bot" ]
