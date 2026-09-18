# ProjectWikit container image. It carries no PostgreSQL of its own; the compose
# file in docker/ runs the official one next to it.
#
#   docker build -t pwikit --build-arg VERSION=v1.0.0 .
#   docker build --target binary --output dist .

FROM node:25-bookworm AS frontend
WORKDIR /src/frontend
COPY frontend/package.json frontend/yarn.lock ./
RUN yarn install --frozen-lockfile
COPY frontend/ ./
COPY static/ /src/static/
RUN yarn build

FROM golang:1.26-bookworm AS build
RUN curl -sSf https://sh.rustup.rs | sh -s -- -y --profile minimal
ENV PATH=/root/.cargo/bin:$PATH
WORKDIR /src
COPY ftml/ ftml/
COPY ftml-capi/ ftml-capi/
RUN cd ftml-capi && cargo build --release
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /src/static/ static/
COPY --from=frontend /src/frontend/node_modules/ frontend/node_modules/
ARG VERSION=v0.0.0-dev
RUN CGO_ENABLED=1 go build -tags assets -trimpath \
      -ldflags "-s -w -X github.com/WikitTeam/ProjectWikit/internal/version.Release=${VERSION}" \
      -o /out/pwikit ./cmd/pwikit \
 && go run ./tools/notices -tags assets -version "${VERSION}" -out /out/NOTICE \
 && cp LICENSE /out/LICENSE \
 && mkdir -p /out/data

FROM scratch AS binary
COPY --from=build /out/pwikit /pwikit

FROM gcr.io/distroless/cc-debian12:nonroot
LABEL org.opencontainers.image.licenses="AGPL-3.0-or-later"
COPY --from=build /out/pwikit /usr/local/bin/pwikit
COPY --from=build /out/LICENSE /out/NOTICE /usr/share/doc/pwikit/
COPY --from=build --chown=65532:65532 /out/data /data
ENV PWIKIT_DATA_DIR=/data \
    PWIKIT_CONTAINER=1
WORKDIR /data
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/pwikit"]
CMD ["serve", "-listen", ":8080"]
