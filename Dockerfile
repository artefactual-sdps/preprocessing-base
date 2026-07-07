# syntax = docker/dockerfile:1.4

ARG GO_VERSION

FROM golang:${GO_VERSION}-alpine AS build-go
WORKDIR /src
ENV CGO_ENABLED=0
COPY --link go.* ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY --link . .

FROM build-go AS build-custom-enduro-worker
ARG VERSION_PATH
ARG VERSION_LONG
ARG VERSION_SHORT
ARG VERSION_GIT_HASH
ARG STRIP=1
RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
	ldflags="-X '${VERSION_PATH}.Long=${VERSION_LONG}' -X '${VERSION_PATH}.Short=${VERSION_SHORT}' -X '${VERSION_PATH}.GitCommit=${VERSION_GIT_HASH}'" && \
	if [ "$STRIP" = "1" ]; then ldflags="-s $ldflags"; fi && \
	go build \
	-trimpath \
	-ldflags="$ldflags" \
	-o /out/custom-enduro-worker \
	./cmd/worker

FROM alpine:3.18.2 AS base
ARG USER_ID=1000
ARG GROUP_ID=1000
RUN addgroup -g ${GROUP_ID} -S enduro
RUN adduser -u ${USER_ID} -S -D enduro enduro
USER enduro
RUN mkdir /home/enduro/shared

FROM base AS custom-enduro-worker
COPY --from=build-custom-enduro-worker --link /out/custom-enduro-worker /home/enduro/bin/custom-enduro-worker
CMD ["/home/enduro/bin/custom-enduro-worker"]
