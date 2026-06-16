# Published for tests as trevex/ocm-tools:v0.1.X
FROM alpine:3@sha256:f5064d3e5f88c467c714509f491853ab2d951932c5cad699c0cb969dcec6f3b4

RUN apk add --no-cache bash sed jq curl
RUN curl -s https://ocm.software/install.sh | bash

ENTRYPOINT ["bash"]


