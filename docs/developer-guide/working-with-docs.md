# Documentation Setup

The documentation of the ARC project is written primarily using Markdown.
All documentation related content can be found in <https://github.com/opendefensecloud/artifact-conduit/tree/main/docs>.
New content also should be added there.

To render the documentation with `mkdocs` locally you have to

- once or when you alter the [Dockerfile](https://github.com/opendefensecloud/artifact-conduit/blob/main/mkdocs.Dockerfile) you need to rebuild the custom container image with `make docs-docker-build`
- run `make docs` to render the documentation

**Build image**

```bash
$ make docs-docker-build
make docs-docker-build 
[+] Building 1.3s (8/8) FINISHED                                                                                             docker:desktop-linux
 => [internal] load build definition from mkdocs.Dockerfile                                                                                  0.0s
 => => transferring dockerfile: 253B                                                                                                         0.0s
 => [internal] load metadata for docker.io/squidfunk/mkdocs-material:latest                                                                  0.0s
 => [internal] load .dockerignore                                                                                                            0.0s
 => => transferring context: 2B                                                                                                              0.0s
 => [1/4] FROM docker.io/squidfunk/mkdocs-material:latest@sha256:146fe500ceaa78c776545f04b9e225220fe0302ba083b5ec7b410ee4ad84bd33            0.0s
 => => resolve docker.io/squidfunk/mkdocs-material:latest@sha256:146fe500ceaa78c776545f04b9e225220fe0302ba083b5ec7b410ee4ad84bd33            0.0s
 => [2/4] RUN pip install mkdocs-glightbox                                                                                                   0.4s
 => [3/4] RUN pip install mkdocs-include-markdown-plugin                                                                                     0.3s
 => [4/4] RUN pip install mkdocs-panzoom-plugin                                                                                              0.4s
 => exporting to image                                                                                                                       0.1s
 => => exporting layers                                                                                                                      0.0s
 => => exporting manifest sha256:d54c58f65c07dcc823ba0b4a09432ac9c0abdae8a73da0fbd5249b982e3bd949                                            0.0s
 => => exporting config sha256:11b5c58174556e5dd40fda25e9c0b105f9354877b3489b4a2a25feb323db3c5f                                              0.0s
 => => exporting attestation manifest sha256:b2d1161a507102c2f37f071caa9d5b3d75dae7b81a0dffbe8e06f027b7bf9ba4                                0.0s
 => => exporting manifest list sha256:fd237be53d68463c7655630fe88bf0119d2377d3e06fb9d574fd1e555bc05801                                       0.0s
 => => naming to docker.io/squidfunk/mkdocs-material:latest                                                                                  0.0s
 => => unpacking to docker.io/squidfunk/mkdocs-material:latest
```

**Render documentation**

```bash
$ make docs
mkdocs serve
INFO    -  Building documentation...
INFO    -  Cleaning site directory
INFO    -  Documentation built in 0.22 seconds
INFO    -  [13:29:25] Watching paths for changes: 'docs', 'mkdocs.yml'
INFO    -  [13:29:25] Serving on http://127.0.0.1:8000/
```

Open <http://127.0.0.1:8000/> to view the documentation with live reloading.
