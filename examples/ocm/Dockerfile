# Published for tests as trevex/ocm-tools:v0.1.X
FROM alpine:3

RUN apk add --no-cache bash sed jq curl
RUN curl -s https://ocm.software/install.sh | bash

ENTRYPOINT ["bash"]


