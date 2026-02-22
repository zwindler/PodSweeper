# PodSweeper Player Terminal
# A minimal container with kubectl for playing PodSweeper
#
# Players exec into this pod to interact with the game using
# the restricted 'player' ServiceAccount.
#
# To customize, use this as a base image or replace entirely:
#   FROM ghcr.io/zwindler/podsweeper-player-terminal:latest
#   RUN apk add --no-cache your-favorite-tools

FROM alpine:3.21

ARG KUBECTL_VERSION=1.32.0
ARG TARGETARCH=amd64

# Install essential tools
RUN apk add --no-cache \
    curl \
    bash \
    jq \
    ca-certificates \
    && rm -rf /var/cache/apk/*

# Install kubectl
RUN curl -LO "https://dl.k8s.io/release/v${KUBECTL_VERSION}/bin/linux/${TARGETARCH}/kubectl" \
    && chmod +x kubectl \
    && mv kubectl /usr/local/bin/

# Create non-root user for the player
RUN adduser -D -s /bin/bash player
USER player
WORKDIR /home/player

# Copy welcome message and shell configuration
COPY --chown=player:player motd /etc/motd
COPY --chown=player:player bashrc /home/player/.bashrc

# Keep the container running
CMD ["sleep", "infinity"]
