# Published for tests as trevex/ocm-tools:v0.1.X
FROM alpine:3@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11

RUN apk add --no-cache bash sed jq curl
RUN curl -s https://ocm.software/install.sh | bash

ENTRYPOINT ["bash"]


