#!/usr/bin/env bash

set -euo pipefail

readonly APP_NAME="dnsmasq-web"
readonly ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly DIST_DIR="${ROOT_DIR}/dist"
readonly DEBUG_DIR="${DIST_DIR}/debug"
readonly RELEASE_DIR="${DIST_DIR}/release"
readonly TOOLCHAINS_DIR="${DNSMASQ_WEB_TOOLCHAINS_DIR:-${ROOT_DIR}/.toolchains}"
readonly GO_IMAGE="${DNSMASQ_WEB_GO_IMAGE:-golang:1.23.12-bookworm}"
readonly GO_IMAGE_TAG="$(printf '%s' "${GO_IMAGE}" | tr '/:@' '---')"
readonly RELEASE_IMAGE_NAME="${DNSMASQ_WEB_RELEASE_IMAGE:-dnsmasq-web-release:${GO_IMAGE_TAG}}"
readonly RELEASE_DOCKERFILE="${ROOT_DIR}/scripts/release.Dockerfile"
readonly GO_BUILD_CACHE_VOLUME="${DNSMASQ_WEB_GO_BUILD_CACHE_VOLUME:-dnsmasq-web-go-build-cache}"
readonly GO_MOD_CACHE_VOLUME="${DNSMASQ_WEB_GO_MOD_CACHE_VOLUME:-dnsmasq-web-go-mod-cache}"
readonly MUSL_BASE_URL="${DNSMASQ_WEB_MUSL_BASE_URL:-https://musl.cc}"

readonly RELEASE_TARGETS=(
	"linux-amd64"
	"linux-arm64"
	"linux-amd64-musl"
	"linux-arm64-musl"
)

usage() {
	cat <<'EOF'
Usage: scripts/build.sh <debug|release|prepare-release-image|release-inner|clean>

debug         Build an unstripped host-architecture binary for local debugging.
release       Build the supported Linux release binaries, using Docker by default.
prepare-release-image
              Build the reusable local Docker image used for release builds.
release-inner Internal entrypoint used by the Dockerized release build.
clean         Remove generated build artifacts.

Environment:
  BUILD_VERSION        Override the ldflags version string.
  RELEASE_LABEL        Override the artifact label used in file names.
  BUILD_USE_CONTAINER  Set to 0 to run release builds on the current host.
  DNSMASQ_WEB_GO_IMAGE Override the Go container image used for release builds.
  DNSMASQ_WEB_PREPARED_RELEASE_IMAGE
                       Set to 0 to skip the reusable local release image.
EOF
}

use_prepared_release_image() {
	case "${DNSMASQ_WEB_PREPARED_RELEASE_IMAGE:-auto}" in
		1|true|TRUE|yes|YES)
			return 0
			;;
		0|false|FALSE|no|NO)
			return 1
			;;
		auto|AUTO|"")
			[[ -z "${CI:-}" ]]
			return
			;;
		*)
			printf 'invalid DNSMASQ_WEB_PREPARED_RELEASE_IMAGE value: %s\n' "${DNSMASQ_WEB_PREPARED_RELEASE_IMAGE}" >&2
			return 1
			;;
	esac
}

build_version() {
	local describe stamp
	describe="$(git -C "${ROOT_DIR}" describe --tags --always --dirty 2>/dev/null || git -C "${ROOT_DIR}" rev-parse --short HEAD)"
	stamp="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
	printf '%s # %s\n' "${describe}" "${stamp}"
}

release_label() {
	local raw
	raw="${RELEASE_LABEL:-$(git -C "${ROOT_DIR}" describe --tags --always --dirty 2>/dev/null || git -C "${ROOT_DIR}" rev-parse --short HEAD)}"
	raw="${raw//\//-}"
	raw="${raw// /-}"
	printf '%s\n' "${raw}"
}

artifact_name() {
	local target="$1"
	printf '%s_%s_%s\n' "${APP_NAME}" "$(release_label)" "${target}"
}

release_ldflags() {
	local target="$1"
	local version
	version="${BUILD_VERSION:-$(build_version)}"
	local flags="-s -w -X 'main.Version=${version}'"
	if [[ "${target}" == *-musl ]]; then
		flags+=" -extldflags '-static -fpic'"
	fi
	printf '%s\n' "${flags}"
}

debug_ldflags() {
	printf '%s\n' "-X 'main.Version=${BUILD_VERSION:-$(build_version)}'"
}

ensure_git_safe_directory() {
	if command -v git >/dev/null 2>&1; then
		git config --global --add safe.directory "${ROOT_DIR}" >/dev/null 2>&1 || true
	fi
}

ensure_host_release_toolchains() {
	if command -v gcc >/dev/null 2>&1 && \
		command -v aarch64-linux-gnu-gcc >/dev/null 2>&1 && \
		[[ -x "${TOOLCHAINS_DIR}/x86_64-linux-musl-cross/bin/x86_64-linux-musl-gcc" ]] && \
		[[ -x "${TOOLCHAINS_DIR}/aarch64-linux-musl-cross/bin/aarch64-linux-musl-gcc" ]]; then
		ensure_git_safe_directory
		return
	fi

	ensure_git_safe_directory
	apt-get update
	DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
		ca-certificates \
		curl \
		gcc \
		gcc-aarch64-linux-gnu \
		libc6-dev \
		libc6-dev-arm64-cross \
		git
	rm -rf /var/lib/apt/lists/*

	ensure_musl_toolchain "x86_64-linux-musl-cross"
	ensure_musl_toolchain "aarch64-linux-musl-cross"
}

ensure_musl_toolchain() {
	local prefix="$1"
	local compiler="${prefix}-gcc"
	local archive="${prefix}.tgz"
	local archive_path="/tmp/${archive}"
	local install_dir="${TOOLCHAINS_DIR}/${prefix}"

	if [[ -x "${install_dir}/bin/${compiler}" ]]; then
		return
	fi

	mkdir -p "${install_dir}"
	curl -fsSL "${MUSL_BASE_URL}/${archive}" -o "${archive_path}"
	tar -xzf "${archive_path}" --strip-components 1 -C "${install_dir}"
	rm -f "${archive_path}"
}

prepare_release_image() {
	docker build \
		--build-arg GO_IMAGE="${GO_IMAGE}" \
		--build-arg MUSL_BASE_URL="${MUSL_BASE_URL}" \
		-t "${RELEASE_IMAGE_NAME}" \
		-f "${RELEASE_DOCKERFILE}" \
		"${ROOT_DIR}"
}

build_debug() {
	mkdir -p "${DEBUG_DIR}"
	(
		cd "${ROOT_DIR}"
		CGO_ENABLED=1 go build \
			-o "${DEBUG_DIR}/${APP_NAME}" \
			-ldflags "$(debug_ldflags)" \
			.
	)
}

target_cc() {
	case "$1" in
		linux-amd64)
			printf '%s\n' gcc
			;;
		linux-arm64)
			printf '%s\n' aarch64-linux-gnu-gcc
			;;
		linux-amd64-musl)
			printf '%s\n' "${TOOLCHAINS_DIR}/x86_64-linux-musl-cross/bin/x86_64-linux-musl-gcc"
			;;
		linux-arm64-musl)
			printf '%s\n' "${TOOLCHAINS_DIR}/aarch64-linux-musl-cross/bin/aarch64-linux-musl-gcc"
			;;
		*)
			printf 'unsupported target: %s\n' "$1" >&2
			return 1
			;;
	esac
}

target_goarch() {
	case "$1" in
		linux-amd64|linux-amd64-musl)
			printf '%s\n' amd64
			;;
		linux-arm64|linux-arm64-musl)
			printf '%s\n' arm64
			;;
		*)
			printf 'unsupported target: %s\n' "$1" >&2
			return 1
			;;
	esac
}

build_release_target() {
	local target="$1"
	local out_path="${RELEASE_DIR}/$(artifact_name "${target}")"
	mkdir -p "${RELEASE_DIR}"
	(
		cd "${ROOT_DIR}"
		CGO_ENABLED=1 \
		GOOS=linux \
		GOARCH="$(target_goarch "${target}")" \
		CC="$(target_cc "${target}")" \
		go build \
			-trimpath \
			-o "${out_path}" \
			-ldflags "$(release_ldflags "${target}")" \
			.
	)
}

fix_permissions() {
	if [[ -n "${HOST_UID:-}" && -n "${HOST_GID:-}" ]]; then
		chown -R "${HOST_UID}:${HOST_GID}" "${DIST_DIR}"
	fi
}

build_release_inner() {
	ensure_host_release_toolchains
	rm -rf "${RELEASE_DIR}"
	mkdir -p "${RELEASE_DIR}"
	for target in "${RELEASE_TARGETS[@]}"; do
		build_release_target "${target}"
	done
	fix_permissions
}

build_release() {
	if [[ "${BUILD_USE_CONTAINER:-1}" != "0" && -z "${DNSMASQ_WEB_BUILD_IN_CONTAINER:-}" ]]; then
		local version label
		local image_name
		local -a docker_args
		version="${BUILD_VERSION:-$(build_version)}"
		label="$(release_label)"
		docker_args=(run --rm \
			-e DNSMASQ_WEB_BUILD_IN_CONTAINER=1 \
			-e BUILD_VERSION="${version}" \
			-e RELEASE_LABEL="${label}" \
			-e HOST_UID="$(id -u)" \
			-e HOST_GID="$(id -g)" \
			-v "${ROOT_DIR}:/workspace" \
			-w /workspace)

		if [[ -z "${CI:-}" ]]; then
			docker_args+=(
				-v "${GO_MOD_CACHE_VOLUME}:/go/pkg/mod"
				-v "${GO_BUILD_CACHE_VOLUME}:/root/.cache/go-build"
			)
		fi

		if use_prepared_release_image; then
			prepare_release_image
			image_name="${RELEASE_IMAGE_NAME}"
			docker_args+=(
				-e DNSMASQ_WEB_TOOLCHAINS_DIR=/opt/dnsmasq-web-toolchains
			)
		else
			image_name="${GO_IMAGE}"
			docker_args+=(
				-e DNSMASQ_WEB_MUSL_BASE_URL="${MUSL_BASE_URL}"
			)
		fi

		docker "${docker_args[@]}" \
			"${image_name}" \
			bash scripts/build.sh release-inner
		return
	fi

	build_release_inner
}

clean() {
	rm -rf "${DIST_DIR}"
}

main() {
	case "${1:-debug}" in
		debug)
			build_debug
			;;
		release)
			build_release
			;;
		prepare-release-image)
			prepare_release_image
			;;
		release-inner)
			build_release_inner
			;;
		clean)
			clean
			;;
		-h|--help|help)
			usage
			;;
		*)
			usage >&2
			return 1
			;;
	esac
}

main "$@"
