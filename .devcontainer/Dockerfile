FROM debian:bookworm

ARG USERNAME=vscode
ARG USER_UID=1000
ARG USER_GID=$USER_UID
RUN groupadd --gid $USER_GID $USERNAME \
    && useradd --uid $USER_UID --gid $USER_GID -m $USERNAME

# ============================================
# === ROOT INSTALLATIONS AND CONFIGURATION ===
# ============================================

RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install --no-install-recommends \
    ca-certificates \
    curl \
    gnupg \
    minisign \
    zsh \
    # === git deps start ===
    git \
    openssh-client \
    # === git deps end ===
    # === cgo deps start ===
    g++ \
    gcc \
    libc6-dev \
    make \
    pkg-config \
    # === cgo deps end ===
    && apt-get clean && rm -rf /var/lib/apt/lists/*

# === GOLANG ===
ENV GOPATH=/go
ENV PATH=$GOPATH/bin:/usr/local/go/bin:$PATH:/home/${USERNAME}/workspace/github.com/slightly-inconvenient/murl/.bin
# Don't autoupgrade go toolchain
ENV GOTOOLCHAIN=local

COPY .devcontainer/checksums/golang.txt /home/${USERNAME}/.checksums/golang.txt
ARG GOLANG_VERSION=1.23.2
RUN curl -OLf "https://go.dev/dl/go${GOLANG_VERSION}.linux-$(dpkg --print-architecture).tar.gz" && \
    sha256sum --check --status --ignore-missing /home/${USERNAME}/.checksums/golang.txt  && \
    tar -xvzf go${GOLANG_VERSION}.linux-$(dpkg --print-architecture).tar.gz -C "/usr/local" && \
    go version && \
    rm go${GOLANG_VERSION}.linux-$(dpkg --print-architecture).tar.gz
RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 1777 "$GOPATH"

# === CONFIGURATION ===
ENV SHELL=/bin/zsh
COPY .bazelversion /home/${USERNAME}/.bazelversion
COPY .devcontainer/.zshrc /home/${USERNAME}/.zshrc

# === BAZELISK ===
COPY .devcontainer/checksums/bazelisk.txt /home/${USERNAME}/.checksums/bazelisk.txt
ARG BAZELISK_VERSION="1.22.0"
RUN curl -OLf "https://github.com/bazelbuild/bazelisk/releases/download/v${BAZELISK_VERSION}/bazelisk-$(go env GOOS)-$(go env GOARCH)" && \
    sha256sum --check --status --ignore-missing /home/${USERNAME}/.checksums/bazelisk.txt && \
    mv bazelisk-$(go env GOOS)-$(go env GOARCH) /usr/local/bin/bazel && \
    chmod +x /usr/local/bin/bazel && \
    BAZELISK_HOME=/home/${USERNAME}/.cache/bazelisk USE_BAZEL_VERSION=$(cat /home/${USERNAME}/.bazelversion) bazel --version && \
    chown -R ${USER_UID}:${USER_GID} /home/${USERNAME}/.cache

# Assign ownership of relevant files/directories to non-root user used to run the image
# This is required as the files otherwise end up as root-owned on the container filesystem
# and thus break when being used by the non-root user
RUN chown ${USER_UID}:${USER_GID} /home/${USERNAME}/.zshrc && \
    chown -R ${USER_UID}:${USER_GID} /go

# ==================================================
# === USERSPACE  INSTALLATIONS AND CONFIGURATION ===
# ==================================================

USER ${USERNAME}

# === GOLANG PACKAGES ===
RUN ls -la /go/ && go install golang.org/x/tools/gopls@v0.16.0 && \
    go install mvdan.cc/gofumpt@v0.6.0

# === ENVIRONMENT SETUP === 
ENV HOME=/home/${USERNAME}

# override editor to make e.g. git work automatically. 
# We know the devcontainer has a working vscode.
ENV EDITOR="code --wait"

# === ZSH ===
RUN curl -fsSL "https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/48ccc7b36de8efb2bd7beb9bd6e0a6f6fe03b95d/tools/install.sh" | zsh && \
    rm /home/${USERNAME}/.zshrc && \
    mv /home/${USERNAME}/.zshrc.pre-oh-my-zsh /home/${USERNAME}/.zshrc