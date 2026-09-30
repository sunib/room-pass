# Multi-stage Dockerfile for Room Pass
#
# Taken from github.com/sunib/voter's devcontainer, where Room Pass started, and
# cut down to what this repository builds and tests: one Go module, its CRDs,
# a k3d fixture with Dex and Traefik, and a Playwright suite. Tool versions are
# kept in step with Voter's so the two containers agree on what each tool means.
#
# Stage 1: CI base image with essential build tools
# Pinned by digest (Scorecard "pinned dependencies"); Dependabot's docker
# ecosystem keeps version + digest current together.
FROM golang:1.27.1-bookworm@sha256:648f440f42a0958804efb24df176f806f9d353b41f1c0627f666428e40310f6b AS ci

# Avoid warnings by switching to noninteractive
ENV DEBIAN_FRONTEND=noninteractive

# Run RUN steps under bash with pipefail so a failure anywhere in a
# `curl ... | tar` download pipeline aborts the build instead of silently
# installing a truncated or empty binary.
SHELL ["/bin/bash", "-o", "pipefail", "-c"]

# Install essential packages + Docker CLI (for DooD via socket mount in CI)
RUN apt-get update \
    && apt-get -y install --no-install-recommends \
    git \
    curl \
    ca-certificates \
    vim \
    less \
    jq \
    yq \
    sudo \
    acl \
    rsync \
    gettext-base \
    gnupg \
    openssl \
    python3 \
    python3-yaml \
    && install -m 0755 -d /etc/apt/keyrings \
    && curl -fsSL https://download.docker.com/linux/debian/gpg -o /etc/apt/keyrings/docker.asc \
    && chmod a+r /etc/apt/keyrings/docker.asc \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian $(. /etc/os-release && echo "$VERSION_CODENAME") stable" \
       > /etc/apt/sources.list.d/docker.list \
    && curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
       -o /etc/apt/keyrings/githubcli-archive-keyring.gpg \
    && chmod a+r /etc/apt/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" \
       > /etc/apt/sources.list.d/github-cli.list \
    && apt-get update \
    && apt-get -y install --no-install-recommends docker-ce-cli docker-buildx-plugin gh \
    && apt-get autoremove -y \
    && apt-get clean -y \
    && rm -rf /var/lib/apt/lists/*

# Tool versions - centralized for easy updates.
# This devcontainer currently supports amd64 tool downloads only.

# Finding the latest versions:
# KUBECTL_VERSION -> https://github.com/kubernetes/kubernetes/releases
# KUSTOMIZE_VERSION -> https://github.com/kubernetes-sigs/kustomize/releases
# GOLANGCI_LINT_VERSION -> https://github.com/golangci/golangci-lint/releases
# K3D_VERSION -> https://github.com/k3d-io/k3d
# TASK_VERSION -> https://github.com/go-task/task/releases
# ACTIONLINT_VERSION -> https://github.com/rhysd/actionlint/releases
# HADOLINT_VERSION -> https://github.com/hadolint/hadolint/releases
# NODE_MAJOR -> https://github.com/nodejs/node/releases (track a current LTS line)
# MARKDOWNLINT_CLI2_VERSION -> https://www.npmjs.com/package/markdownlint-cli2?activeTab=versions

ENV PATH="/go/bin:/usr/local/go/bin:${PATH}" \
    KUBECTL_VERSION=v1.37.0 \
    KUSTOMIZE_VERSION=5.8.1 \
    GOLANGCI_LINT_VERSION=v2.13.2 \
    K3D_VERSION=v5.9.0 \
    TASK_VERSION=v3.53.1 \
    ACTIONLINT_VERSION=1.7.12 \
    HADOLINT_VERSION=2.15.1 \
    NODE_MAJOR=24 \
    MARKDOWNLINT_CLI2_VERSION=0.23.3

# Fail early on unsupported architectures instead of producing a partial image.
RUN test "$(dpkg --print-architecture)" = "amd64" \
    || (echo "This devcontainer currently supports amd64 only." && exit 1)

# Install kubectl
RUN curl -LO "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl" \
    && chmod +x kubectl \
    && mv kubectl /usr/local/bin/

# Install Kustomize
RUN curl -Lo kustomize.tar.gz "https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv${KUSTOMIZE_VERSION}/kustomize_v${KUSTOMIZE_VERSION}_linux_amd64.tar.gz" && \
    tar -zxvf kustomize.tar.gz && \
    mv kustomize /usr/local/bin/ && \
    rm kustomize.tar.gz

# Install golangci-lint
RUN curl -sSfL https://golangci-lint.run/install.sh \
    | sh -s -- -b /usr/local/bin ${GOLANGCI_LINT_VERSION}

# Install k3d (faster and kind alternative, also more stable with our docker-outside-docker setup)
RUN curl -fsSL -o /usr/local/bin/k3d "https://github.com/k3d-io/k3d/releases/download/${K3D_VERSION}/k3d-linux-amd64" \
    && chmod +x /usr/local/bin/k3d

# Install Task
RUN curl -fsSL "https://github.com/go-task/task/releases/download/${TASK_VERSION}/task_linux_amd64.tar.gz" \
    | tar -xzO task > /usr/local/bin/task \
    && chmod +x /usr/local/bin/task

# Install actionlint (static linter for GitHub Actions workflows)
RUN asset="actionlint_${ACTIONLINT_VERSION}_linux_amd64.tar.gz" \
    && base="https://github.com/rhysd/actionlint/releases/download/v${ACTIONLINT_VERSION}" \
    && tmpdir="$(mktemp -d)" \
    && curl -fsSL "${base}/${asset}" -o "${tmpdir}/${asset}" \
    && curl -fsSL "${base}/actionlint_${ACTIONLINT_VERSION}_checksums.txt" -o "${tmpdir}/checksums.txt" \
    && cd "${tmpdir}" \
    && grep "  ${asset}$" checksums.txt | sha256sum -c - \
    && tar -xzf "${asset}" actionlint \
    && install -m 0755 actionlint /usr/local/bin/actionlint \
    && rm -rf "${tmpdir}"

# Install hadolint (static linter for Dockerfiles). Since 2.15.0 the release no
# longer ships a per-asset .sha256 file, only one combined checksums.sha256 in
# `sha256sum *filename` format (space-asterisk, not the two-space format the
# other combined-checksum tools below use).
RUN asset="hadolint-linux-x86_64" \
    && base="https://github.com/hadolint/hadolint/releases/download/v${HADOLINT_VERSION}" \
    && tmpdir="$(mktemp -d)" \
    && curl -fsSL "${base}/${asset}" -o "${tmpdir}/${asset}" \
    && curl -fsSL "${base}/checksums.sha256" -o "${tmpdir}/checksums.sha256" \
    && cd "${tmpdir}" \
    && grep " \*${asset}$" checksums.sha256 | sha256sum -c - \
    && install -m 0755 "${asset}" /usr/local/bin/hadolint \
    && rm -rf "${tmpdir}"

# Install Node.js (provides npm + npx).
#
# Node runs the Playwright browser suite in test/browser and the markdownlint-cli2
# docs linter installed below. It lives in the ci stage because the browser job
# runs against this stage directly; the dev stage inherits it.
RUN curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key \
       | gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg \
    && chmod a+r /etc/apt/keyrings/nodesource.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_${NODE_MAJOR}.x nodistro main" \
       > /etc/apt/sources.list.d/nodesource.list \
    && apt-get update \
    && apt-get -y install --no-install-recommends nodejs \
    && apt-get autoremove -y \
    && apt-get clean -y \
    && rm -rf /var/lib/apt/lists/*

# Install markdownlint-cli2 (structural markdown linter: fences, headings, lists,
# tables). Ships as an npm package only -- there is no standalone binary release --
# so it is installed globally with the version pinned in the ENV block above.
RUN npm install -g "markdownlint-cli2@${MARKDOWNLINT_CLI2_VERSION}" \
    && npm cache clean --force

# Set working directory
WORKDIR /workspaces

# Create godev group for shared Go development directory access
# This allows both root (during build) and vscode user (during dev) to write to /go
RUN groupadd --gid 2000 godev

# Set group permissions on /go directory for shared access
# This MUST be done BEFORE any go mod download or go install operations
RUN chgrp -R godev /go && \
    chmod -R 2775 /go && \
    setfacl -d -m g:godev:rwx /go && \
    setfacl -d -m u::rwx /go && \
    setfacl -d -m o::rx /go

# Install Go tools used by the project (using @version doesn't need go.mod)
# These run AFTER permissions are set to ensure proper group ownership.
#
# Retried, unlike the pinned single-binary downloads above. This layer resolves
# hundreds of modules through proxy.golang.org and verifies each against
# sum.golang.org, so it has far more chances to meet a transient network fault:
# CI run 34372040435 died here on an HTTP/2 stream error from the checksum
# database ("reading https://sum.golang.org/tile/...: INTERNAL_ERROR"). A cold
# CI image build reaches this point several minutes in, and a flake here fails
# the whole pipeline before a single test runs.
#
# Three attempts with a growing pause; an exhausted loop exits non-zero rather
# than falling through, so a genuinely broken install still fails the build
# instead of producing an image that is missing controller-gen.
RUN set -e; \
    install_go_tools() { \
        go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.22.0 \
        && go install sigs.k8s.io/controller-runtime/tools/setup-envtest@v0.25.0 \
        && go install golang.org/x/tools/cmd/goimports@v0.49.0; \
    }; \
    for attempt in 1 2 3; do \
        if install_go_tools; then exit 0; fi; \
        echo "go install attempt ${attempt}/3 failed; retrying" >&2; \
        sleep $((attempt * 15)); \
    done; \
    echo "go install failed after 3 attempts" >&2; \
    exit 1

# Initialize golangci-lint cache by running it once on an empty directory
# This downloads linter dependencies without needing source code
RUN mkdir -p /tmp/golangci-init && cd /tmp/golangci-init \
    && go mod init example.com/init \
    && printf 'package main\n\nfunc main() {}\n' > main.go \
    && golangci-lint run --timeout=5m || true \
    && cd / && rm -rf /tmp/golangci-init

# Pre-download Go modules for caching - placed AFTER tool installation
# This layer will be cached and only rebuilt when go.mod/go.sum changes
# Moving this down prevents tool reinstallation when dependencies change
# The single-quoted lines below are shell written verbatim into the profile
# script; build-time expansion must NOT happen, so single quotes are intended.
# hadolint ignore=SC2016
RUN printf '%s\n' \
    'export PATH="/go/bin:/usr/local/go/bin:${PATH}"' \
    > /etc/profile.d/go-toolchain-path.sh \
    && chmod 0644 /etc/profile.d/go-toolchain-path.sh \
    && printf '%s\n' \
    '# Silently load the optional repo-root .env into login shells.' \
    'workspace_dir="${PROJECT_PATH:-}"' \
    'if [ -n "${workspace_dir}" ] && [ -f "${workspace_dir}/.env" ]; then' \
    '  set -a' \
    '  . "${workspace_dir}/.env"' \
    '  set +a' \
    'fi' \
    > /etc/profile.d/workspace-dotenv.sh \
    && chmod 0644 /etc/profile.d/workspace-dotenv.sh

# Dependency warm-up, into a scratch directory rather than WORKDIR so the
# workspace bind mount cannot hide it. What it populates is /go/pkg/mod.
COPY go.mod go.sum /tmp/warm/
RUN cd /tmp/warm && go mod download && rm -rf /tmp/warm

# Switch back to dialog for any ad-hoc use of apt-get
ENV DEBIAN_FRONTEND=dialog

# Default command
CMD ["/bin/bash"]

# Stage 2: Development container, used for local development (https://github.com/devcontainers/spec)
FROM ci AS dev

USER root

# Switch to noninteractive for package installation
ENV DEBIAN_FRONTEND=noninteractive

# Same pipefail hardening as the ci stage (SHELL is not inherited across stages).
SHELL ["/bin/bash", "-o", "pipefail", "-c"]

# Dev-only tool versions - centralized for easy updates.
# These tools are only used inside the devcontainer (local development, IDE
# tooling, debugging), never in CI, so they live in the dev stage.
#
# Finding the latest versions:
# DLV_VERSION -> https://github.com/go-delve/delve/releases
# GOPLS_VERSION -> https://github.com/golang/tools/releases
# STATICCHECK_VERSION -> https://github.com/dominikh/go-tools/releases
ENV DLV_VERSION=v1.27.1 \
    GOPLS_VERSION=v0.23.0 \
    STATICCHECK_VERSION=v0.8.1

# k3d is already installed in the ci stage above.
RUN apt-get update \
    && apt-get -y install --no-install-recommends bash-completion \
    && apt-get autoremove -y \
    && apt-get clean -y \
    && rm -rf /var/lib/apt/lists/*

# Enable bash completion and auto-load Task completions for interactive shells in the dev image only.
# Single-quoted lines are written verbatim into bash.bashrc; no build-time expansion intended.
# hadolint ignore=SC2016
RUN printf '%s\n' \
    '' \
    '# Enable bash completion and Task completions in interactive shells.' \
    'if ! shopt -oq posix; then' \
    '  if [ -f /usr/share/bash-completion/bash_completion ]; then' \
    '    . /usr/share/bash-completion/bash_completion' \
    '  elif [ -f /etc/bash_completion ]; then' \
    '    . /etc/bash_completion' \
    '  fi' \
    '  if command -v task >/dev/null 2>&1 && ! complete -p task >/dev/null 2>&1; then' \
    '    eval "$(task --completion bash)"' \
    '  fi' \
    'fi' \
    >> /etc/bash.bashrc

# Install Delve debugger for Go debugging in VSCode
# ACLs from CI stage should be preserved, so no need to re-apply
RUN go install github.com/go-delve/delve/cmd/dlv@${DLV_VERSION}

# Install VSCode Go extension tools (gopls and staticcheck). Both are editor-only:
# gopls is the language server, and this standalone staticcheck powers the editor's
# live (as-you-type) diagnostics. Neither is a lint step -- `task lint` already runs
# staticcheck's analyzers via golangci-lint's bundled staticcheck linter (configured in
# .golangci.yml), so this binary is for the IDE, not a second lint pass.
RUN go install golang.org/x/tools/gopls@${GOPLS_VERSION} \
    && go install honnef.co/go/tools/cmd/staticcheck@${STATICCHECK_VERSION}

# Create vscode user for non-root development and add to godev group
RUN groupadd --gid 1000 vscode \
    && useradd --uid 1000 --gid vscode --shell /bin/bash --create-home vscode \
    && echo 'vscode ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers.d/vscode \
    && chmod 0440 /etc/sudoers.d/vscode \
    && usermod -aG godev vscode

# Ensure docker group exists and add vscode user to it
# This works with the docker-in-docker feature
RUN groupadd -f docker && usermod -aG docker vscode

# Ensure vscode user can write to workspace (empty, so fast)
# Note: /go permissions are already set in CI stage and preserved here
RUN chown -R vscode:vscode /workspaces && \
    chmod -R 755 /workspaces

# Switch back to vscode user for development
USER vscode

# Switch back to dialog for any ad-hoc use of apt-get
ENV DEBIAN_FRONTEND=dialog

# Default command
CMD ["/bin/bash"]
