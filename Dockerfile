# SPDX-License-Identifier: Apache-2.0
# SPDX-FileCopyrightText: 2026 The publisher-npm Authors

FROM golang:1.25-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/plugin ./cmd/plugin

FROM node:24-alpine
RUN npm install --global corepack@0.34.0 && \
    corepack enable && \
    corepack install --global pnpm@10.14.0 yarn@1.22.22
COPY --from=build /out/plugin /usr/local/bin/plugin
WORKDIR /workspace
USER node
ENTRYPOINT ["/usr/local/bin/plugin"]
