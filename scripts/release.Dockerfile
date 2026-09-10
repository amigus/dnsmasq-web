ARG GO_IMAGE=golang:1.23.12-bookworm
FROM ${GO_IMAGE}

ARG MUSL_BASE_URL=https://musl.cc
ENV DNSMASQ_WEB_TOOLCHAINS_DIR=/opt/dnsmasq-web-toolchains

RUN set -eux; \
	apt-get update; \
	DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
		ca-certificates \
		curl \
		gcc \
		gcc-aarch64-linux-gnu \
		git \
		libc6-dev \
		libc6-dev-arm64-cross; \
	rm -rf /var/lib/apt/lists/*; \
	mkdir -p \
		"${DNSMASQ_WEB_TOOLCHAINS_DIR}/x86_64-linux-musl-cross" \
		"${DNSMASQ_WEB_TOOLCHAINS_DIR}/aarch64-linux-musl-cross"; \
	curl -fsSL "${MUSL_BASE_URL}/x86_64-linux-musl-cross.tgz" -o /tmp/x86_64-linux-musl-cross.tgz; \
	tar -xzf /tmp/x86_64-linux-musl-cross.tgz --strip-components 1 -C "${DNSMASQ_WEB_TOOLCHAINS_DIR}/x86_64-linux-musl-cross"; \
	rm -f /tmp/x86_64-linux-musl-cross.tgz; \
	curl -fsSL "${MUSL_BASE_URL}/aarch64-linux-musl-cross.tgz" -o /tmp/aarch64-linux-musl-cross.tgz; \
	tar -xzf /tmp/aarch64-linux-musl-cross.tgz --strip-components 1 -C "${DNSMASQ_WEB_TOOLCHAINS_DIR}/aarch64-linux-musl-cross"; \
	rm -f /tmp/aarch64-linux-musl-cross.tgz
