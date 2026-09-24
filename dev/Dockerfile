FROM golang:1.26.8-bookworm AS golang
FROM node:22-bookworm
COPY --from=golang /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:/usr/local/bin:${PATH}"
ENV CGO_ENABLED=1
RUN apt-get update && apt-get install -y --no-install-recommends build-essential git ca-certificates pkg-config ffmpeg \
    && rm -rf /var/lib/apt/lists/*
RUN corepack enable && corepack prepare yarn@1.22.22 --activate
RUN GOBIN=/usr/local/bin go install github.com/air-verse/air@v1.67.4
WORKDIR /src
CMD ["bash"]

RUN apt-get update && apt-get install -y --no-install-recommends libayatana-appindicator3-dev \
    && rm -rf /var/lib/apt/lists/*
