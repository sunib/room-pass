# The Room Pass toolchain: one image for CI and for the devcontainer.
#
#   ci   everything `task ci` needs. GitHub Actions builds this stage and runs
#        `task ci` inside it; nothing is installed on the runner, and nothing is
#        downloaded while the checks run.
#   dev  ci plus editor and debugging tools, and a non-root user.
#
# Every tool is a pinned version fetched from its own release and checked against
# a SHA-256 written here, not against a checksum file fetched next to it. No
# install scripts, no vendor apt repositories, no "latest". Debian packages come
# from the base image's own Bookworm archive.
#
# Versions match the voter devcontainer this was taken from. To bump a tool:
# change its version, download the new asset, check the hash against the
# checksum file the project publishes, and change the hash. Docker's static
# tarball is the exception: download.docker.com publishes no checksum, so its
# hash is the one recorded when the version was first pinned.
FROM golang:1.27.1-bookworm@sha256:69a7b9788769bec032d238959b61854e9ae87f57be9029ec04e9885fabf99195 AS ci

ENV DEBIAN_FRONTEND=noninteractive

# pipefail, so a failed download in a pipeline fails the build instead of
# installing a truncated binary.
SHELL ["/bin/bash", "-o", "pipefail", "-c"]

RUN test "$(dpkg --print-architecture)" = "amd64" \
    || (echo "This image supports amd64 only." && exit 1)

# python3-yaml: test/e2e/up.sh parses the rendered authenticator before handing
# it to the apiserver. openssl: the fixture's TLS and cookie keys.
RUN apt-get update \
    && apt-get -y install --no-install-recommends \
       ca-certificates curl git jq less xz-utils acl sudo \
       openssl python3 python3-yaml \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

ENV DOCKER_VERSION=29.8.1 \
    DOCKER_SHA256=d8db66739d2e28d4933786d73e918d9be643a67fbd835db1bf740d650a259e70 \
    BUILDX_VERSION=0.37.1 \
    BUILDX_SHA256=9447199cdb435f25880548343c128a4b6650e8891ee598905d8d29d39a8e359b \
    KUBECTL_VERSION=1.37.0 \
    KUBECTL_SHA256=6129359f4e1f3848a5572ccb0b26cf28b8ca08cef38c95a765b2f64a2c961a2f \
    KUSTOMIZE_VERSION=5.8.1 \
    KUSTOMIZE_SHA256=029a7f0f4e1932c52a0476cf02a0fd855c0bb85694b82c338fc648dcb53a819d \
    GOLANGCI_LINT_VERSION=2.13.2 \
    GOLANGCI_LINT_SHA256=2277d43b98ec0054280f2ac26b53268bae97682444678a59a657dd565da021d6 \
    K3D_VERSION=5.9.0 \
    K3D_SHA256=06d8f25bc3a971c4eb29e0ff08429b180402db0f4dec838c9eac427e296800a0 \
    TASK_VERSION=3.53.1 \
    TASK_SHA256=a54a408f6861ff921f6e87774180db31bacd8c1e7c944ca696db9fea49a82fc7 \
    ACTIONLINT_VERSION=1.7.12 \
    ACTIONLINT_SHA256=8aca8db96f1b94770f1b0d72b6dddcb1ebb8123cb3712530b08cc387b349a3d8 \
    HADOLINT_VERSION=2.15.1 \
    HADOLINT_SHA256=c7187db94eeeeca956519a6af171adc31453941a1e777961f6e680f697c8c507 \
    NODE_VERSION=24.21.0 \
    NODE_SHA256=fd8e59d5a511510f6a298afb548f18c7d2b1be404d8b4a27d94fbe49f56cb2d6

# fetch URL SHA256 FILE: download, then refuse anything that is not exactly the
# pinned artifact. The single quotes are the script's own text, not expansions.
# hadolint ignore=SC2016
RUN printf '%s\n' '#!/bin/sh' 'set -eu' \
       'curl -fsSL --retry 3 -o "$3" "$1"' \
       'echo "$2  $3" | sha256sum -c -' \
       > /usr/local/bin/fetch \
    && chmod 0755 /usr/local/bin/fetch

WORKDIR /tmp/tools

RUN fetch "https://download.docker.com/linux/static/stable/x86_64/docker-${DOCKER_VERSION}.tgz" "${DOCKER_SHA256}" docker.tgz \
    && tar -xzf docker.tgz --strip-components=1 -C /usr/local/bin docker/docker \
    && mkdir -p /usr/local/lib/docker/cli-plugins \
    && fetch "https://github.com/docker/buildx/releases/download/v${BUILDX_VERSION}/buildx-v${BUILDX_VERSION}.linux-amd64" "${BUILDX_SHA256}" /usr/local/lib/docker/cli-plugins/docker-buildx \
    && chmod 0755 /usr/local/lib/docker/cli-plugins/docker-buildx \
    && fetch "https://dl.k8s.io/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl" "${KUBECTL_SHA256}" /usr/local/bin/kubectl \
    && chmod 0755 /usr/local/bin/kubectl \
    && fetch "https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv${KUSTOMIZE_VERSION}/kustomize_v${KUSTOMIZE_VERSION}_linux_amd64.tar.gz" "${KUSTOMIZE_SHA256}" kustomize.tgz \
    && tar -xzf kustomize.tgz -C /usr/local/bin kustomize \
    && fetch "https://github.com/golangci/golangci-lint/releases/download/v${GOLANGCI_LINT_VERSION}/golangci-lint-${GOLANGCI_LINT_VERSION}-linux-amd64.tar.gz" "${GOLANGCI_LINT_SHA256}" golangci-lint.tgz \
    && tar -xzf golangci-lint.tgz --strip-components=1 -C /usr/local/bin "golangci-lint-${GOLANGCI_LINT_VERSION}-linux-amd64/golangci-lint" \
    && fetch "https://github.com/k3d-io/k3d/releases/download/v${K3D_VERSION}/k3d-linux-amd64" "${K3D_SHA256}" /usr/local/bin/k3d \
    && chmod 0755 /usr/local/bin/k3d \
    && fetch "https://github.com/go-task/task/releases/download/v${TASK_VERSION}/task_linux_amd64.tar.gz" "${TASK_SHA256}" task.tgz \
    && tar -xzf task.tgz -C /usr/local/bin task \
    && fetch "https://github.com/rhysd/actionlint/releases/download/v${ACTIONLINT_VERSION}/actionlint_${ACTIONLINT_VERSION}_linux_amd64.tar.gz" "${ACTIONLINT_SHA256}" actionlint.tgz \
    && tar -xzf actionlint.tgz -C /usr/local/bin actionlint \
    && fetch "https://github.com/hadolint/hadolint/releases/download/v${HADOLINT_VERSION}/hadolint-linux-x86_64" "${HADOLINT_SHA256}" /usr/local/bin/hadolint \
    && chmod 0755 /usr/local/bin/hadolint \
    && fetch "https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-x64.tar.xz" "${NODE_SHA256}" node.txz \
    && tar -xJf node.txz --strip-components=1 -C /usr/local --exclude='*.md' --exclude=LICENSE \
    && rm -rf /tmp/tools

WORKDIR /workspaces

# /go is shared by root (image build) and vscode (devcontainer) through a group.
RUN groupadd --gid 2000 godev \
    && chgrp -R godev /go \
    && chmod -R 2775 /go \
    && setfacl -d -m g:godev:rwx /go \
    && setfacl -d -m u::rwx /go \
    && setfacl -d -m o::rx /go

# Go tools by exact module version; the Go checksum database verifies them.
# Retried: this layer resolves many modules through proxy.golang.org, and one
# transient network fault otherwise fails the whole build.
RUN for attempt in 1 2 3; do \
        go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.22.0 \
        && go install sigs.k8s.io/controller-runtime/tools/setup-envtest@v0.25.0 \
        && go install golang.org/x/tools/cmd/goimports@v0.49.0 \
        && go clean -cache && exit 0; \
        echo "go install attempt ${attempt}/3 failed; retrying" >&2; sleep $((attempt * 15)); \
    done; exit 1

# The API server and etcd test/integration runs against, at one exact version,
# in the image rather than downloaded by every run.
ENV ENVTEST_K8S_VERSION=1.31.0 \
    KUBEBUILDER_ASSETS=/usr/local/kubebuilder/k8s/1.31.0-linux-amd64
RUN setup-envtest use "${ENVTEST_K8S_VERSION}" --bin-dir /usr/local/kubebuilder \
    && test -x "${KUBEBUILDER_ASSETS}/kube-apiserver"

# Chromium and its system libraries for test/browser, at the Playwright version
# test/browser/package-lock.json resolves. Installed from that lockfile, so the
# image and the tests cannot disagree about which browser build they expect.
ENV PLAYWRIGHT_BROWSERS_PATH=/ms-playwright
COPY test/browser/package.json test/browser/package-lock.json /tmp/playwright/
RUN cd /tmp/playwright \
    && npm ci --no-audit --no-fund \
    && npx playwright install --with-deps chromium \
    && chmod -R a+rX "${PLAYWRIGHT_BROWSERS_PATH}" \
    && rm -rf /tmp/playwright /root/.npm \
    && rm -rf /var/lib/apt/lists/*

# Warm golangci-lint's own cache and the module cache. Into scratch directories,
# so the workspace bind mount cannot hide them.
RUN mkdir -p /tmp/golangci-init && cd /tmp/golangci-init \
    && go mod init example.com/init \
    && printf 'package main\n\nfunc main() {}\n' > main.go \
    && (golangci-lint run --timeout=5m || true) \
    && cd / && rm -rf /tmp/golangci-init && go clean -cache
COPY go.mod go.sum /tmp/warm/
RUN cd /tmp/warm && go mod download && rm -rf /tmp/warm

# Login shells (`bash -lc`, which CI uses) keep Go on PATH and load an optional
# repo-root .env.
# hadolint ignore=SC2016
RUN printf '%s\n' 'export PATH="/go/bin:/usr/local/go/bin:${PATH}"' \
       > /etc/profile.d/go-toolchain-path.sh \
    && printf '%s\n' \
       'if [ -n "${PROJECT_PATH:-}" ] && [ -f "${PROJECT_PATH}/.env" ]; then' \
       '  set -a; . "${PROJECT_PATH}/.env"; set +a' \
       'fi' \
       > /etc/profile.d/workspace-dotenv.sh

ENV DEBIAN_FRONTEND=dialog
CMD ["/bin/bash"]

FROM ci AS dev

USER root
ENV DEBIAN_FRONTEND=noninteractive
SHELL ["/bin/bash", "-o", "pipefail", "-c"]

ENV GH_VERSION=2.102.0 \
    GH_SHA256=bb766f710eef8ede859c18578c72c327597cd4c8a85b06001b1f3843c6019386 \
    DLV_VERSION=v1.27.1 \
    GOPLS_VERSION=v0.23.0 \
    STATICCHECK_VERSION=v0.8.1

RUN apt-get update \
    && apt-get -y install --no-install-recommends bash-completion gnupg openssh-client vim \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

RUN fetch "https://github.com/cli/cli/releases/download/v${GH_VERSION}/gh_${GH_VERSION}_linux_amd64.tar.gz" "${GH_SHA256}" /tmp/gh.tgz \
    && tar -xzf /tmp/gh.tgz --strip-components=2 -C /usr/local/bin "gh_${GH_VERSION}_linux_amd64/bin/gh" \
    && rm /tmp/gh.tgz

# Editor and debugger only. gopls is the language server and this staticcheck
# drives as-you-type diagnostics; `task lint` runs golangci-lint, not these.
RUN go install github.com/go-delve/delve/cmd/dlv@${DLV_VERSION} \
    && go install golang.org/x/tools/gopls@${GOPLS_VERSION} \
    && go install honnef.co/go/tools/cmd/staticcheck@${STATICCHECK_VERSION} \
    && go clean -cache

# hadolint ignore=SC2016
RUN printf '%s\n' '' \
       'if ! shopt -oq posix && [ -f /usr/share/bash-completion/bash_completion ]; then' \
       '  . /usr/share/bash-completion/bash_completion' \
       '  command -v task >/dev/null 2>&1 && eval "$(task --completion bash)"' \
       'fi' \
       >> /etc/bash.bashrc

RUN groupadd --gid 1000 vscode \
    && useradd --uid 1000 --gid vscode --shell /bin/bash --create-home vscode \
    && echo 'vscode ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/vscode \
    && chmod 0440 /etc/sudoers.d/vscode \
    && usermod -aG godev vscode \
    && groupadd -f docker && usermod -aG docker vscode \
    && chown -R vscode:vscode /workspaces

USER vscode
ENV DEBIAN_FRONTEND=dialog
CMD ["/bin/bash"]
