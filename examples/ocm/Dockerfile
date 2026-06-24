# Published for tests as trevex/ocm-tools:v0.1.X
FROM alpine:3@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

RUN apk add --no-cache bash sed jq curl
RUN curl -s https://ocm.software/install.sh | bash

ENTRYPOINT ["bash"]


