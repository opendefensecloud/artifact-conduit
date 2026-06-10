# Published for tests as trevex/ocm-tools:v0.1.X
FROM alpine:3@sha256:a2d49ea686c2adfe3c992e47dc3b5e7fa6e6b5055609400dc2acaeb241c829f4

RUN apk add --no-cache bash sed jq curl
RUN curl -s https://ocm.software/install.sh | bash

ENTRYPOINT ["bash"]


