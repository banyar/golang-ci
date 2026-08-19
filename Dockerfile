# Dockerfile for the golangci-lint web dashboard (backend/cmd/dashboard) --
# a separate binary from the main rt-external-api server, see plans/ for
# why. golangci/ is its own Go module (see golangci/go.mod), so this build
# is self-contained -- run it with golangci/ itself as the build context:
#   cd golangci && docker build -t golangci-dashboard .
ARG app_user=golangci
ARG builder_dir=/app
ARG RT_DATACORE_ACCESS_TOKEN
ARG GO_PACKAGES_ACCESS_TOKEN

FROM golang:1.24.0-alpine AS builder

ARG GO_PACKAGES_ACCESS_TOKEN
ARG builder_dir
ARG RT_DATACORE_ACCESS_TOKEN
# Pinned to the exact version every milestone's verification in this
# session used -- plans/2026-08-04-golangci-m2-implementation.md
# found golangci-lint v2's CLI flags differ from v1, so an unpinned
# "latest" here could silently drift the runtime behavior this whole
# suite was verified against.
ARG GOLANGCI_LINT_VERSION=v2.12.2

WORKDIR ${builder_dir}

COPY go.* ./

RUN apk add --no-cache git
RUN git config --global url."https://oauth2:${RT_DATACORE_ACCESS_TOKEN}@git.frontiir.net".insteadOf "https://git.frontiir.net"
RUN git config --global url."https://oauth2:${GO_PACKAGES_ACCESS_TOKEN}@git.frontiir.net/sa-dev/frontiirgopackages".insteadOf "https://git.frontiir.net/sa-dev/frontiirgopackages"
ENV GOPRIVATE=git.frontiir.net
RUN go mod tidy
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o golangci-dashboard ./backend/cmd/dashboard
RUN go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${GOLANGCI_LINT_VERSION}


# Runtime image -- needs git and golangci-lint as actual subprocesses at
# request time (backend/scanner, backend/fixer, backend/worker/rollback.go
# all shell out to them), not just to build the Go binary.
FROM alpine:latest

ARG app_user
ARG builder_dir

ENV APP_WORKDIR="/app"

RUN apk add --no-cache git && adduser -D ${app_user}

WORKDIR ${APP_WORKDIR}

COPY --chown=${app_user}:${app_user} --from=builder ${builder_dir}/golangci-dashboard ./golangci-dashboard
COPY --chown=${app_user}:${app_user} --from=builder ${builder_dir}/backend/config/ ./backend/config/
COPY --chown=${app_user}:${app_user} --from=builder /root/go/bin/golangci-lint /usr/local/bin/golangci-lint

USER ${app_user}

ENTRYPOINT [ "./golangci-dashboard" ]
