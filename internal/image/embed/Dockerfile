FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends git curl ca-certificates wget openssh-client jq \
      python3 python3-pip python3-venv && \
    rm -rf /var/lib/apt/lists/*

RUN mkdir -p -m 755 /etc/apt/keyrings && \
    wget -qO- https://cli.github.com/packages/githubcli-archive-keyring.gpg \
      > /etc/apt/keyrings/githubcli-archive-keyring.gpg && \
    chmod go+r /etc/apt/keyrings/githubcli-archive-keyring.gpg && \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" \
      > /etc/apt/sources.list.d/github-cli.list && \
    apt-get update && \
    apt-get install -y --no-install-recommends gh && \
    rm -rf /var/lib/apt/lists/*

RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - && \
    apt-get install -y --no-install-recommends nodejs && \
    rm -rf /var/lib/apt/lists/* && \
    corepack enable && \
    corepack prepare pnpm@latest --activate

RUN ARCH="$(dpkg --print-architecture)" && \
    if [ "$ARCH" = "amd64" ]; then GOARCH="amd64"; \
    elif [ "$ARCH" = "arm64" ]; then GOARCH="arm64"; \
    else echo "Unsupported architecture: $ARCH" && exit 1; fi && \
    curl -fsSL "https://go.dev/dl/go1.23.6.linux-${GOARCH}.tar.gz" | tar -C /usr/local -xz

# GitLab CLI (glab)
RUN GLAB_VERSION=$(curl -fsSL "https://gitlab.com/api/v4/projects/gitlab-org%2Fcli/releases/permalink/latest" | jq -r '.tag_name' | sed 's/^v//') \
    && curl -fsSL "https://gitlab.com/gitlab-org/cli/-/releases/v${GLAB_VERSION}/downloads/glab_${GLAB_VERSION}_linux_$(dpkg --print-architecture).deb" -o /tmp/glab.deb \
    && dpkg -i /tmp/glab.deb \
    && rm /tmp/glab.deb

RUN useradd -m -s /bin/bash claude

ENV PATH="/usr/local/go/bin:/home/claude/.local/bin:${PATH}"

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
# Create base directory for host path symlinks (owned by claude)
RUN mkdir -p /Users && chown claude:claude /Users

WORKDIR /workspace
# Note: WORKDIR is overridden at runtime via --workdir to match host path

ENTRYPOINT ["/entrypoint.sh"]
CMD ["claude"]
