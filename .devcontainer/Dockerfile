FROM mcr.microsoft.com/devcontainers/go:1.26-bookworm

ARG TARGETARCH
ARG JUST_VERSION=1.58.0

USER root

RUN apt-get update \
    && apt-get install --yes --no-install-recommends ca-certificates curl pipx \
    && rm -rf /var/lib/apt/lists/*

RUN set -eux; \
    case "${TARGETARCH}" in \
      amd64) \
        just_arch="x86_64"; \
        just_sha256="4a5cc2f53e6f0f8c59092a6cc38291eb729d46a7dd95d3ae582008881b84931d" ;; \
      arm64) \
        just_arch="aarch64"; \
        just_sha256="748237128c4c40cbdabc65e841d05ceba13cc23a91eaba395495894c1d9764df" ;; \
      *) echo "unsupported TARGETARCH: ${TARGETARCH}" >&2; exit 1 ;; \
    esac; \
    archive="/tmp/just.tar.gz"; \
    curl --fail --location --retry 3 --show-error \
      "https://github.com/casey/just/releases/download/${JUST_VERSION}/just-${JUST_VERSION}-${just_arch}-unknown-linux-musl.tar.gz" \
      --output "${archive}"; \
    echo "${just_sha256}  ${archive}" | sha256sum --check --strict; \
    tar --extract --gzip --file "${archive}" --directory /usr/local/bin just; \
    chmod 0755 /usr/local/bin/just; \
    rm "${archive}"

USER vscode

ENV PATH="/home/vscode/.local/bin:${PATH}"

RUN pipx install pre-commit==4.6.2
