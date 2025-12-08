# Documentation Changes

Docs help our users understand how to use ARC and fix their own problems.

General guidelines:

* Explain when you would want to use a feature.
* Provide working examples.
* Format code using back-ticks to avoid it being reported as a spelling error.
* Prefer 1 sentence per line of markdown
* Follow the recommendations in the official [Kubernetes Documentation Style Guide](https://kubernetes.io/docs/contribute/style/style-guide/).
    * Particularly useful sections include [Content best practices](https://kubernetes.io/docs/contribute/style/style-guide/#content-best-practices) and [Patterns to avoid](https://kubernetes.io/docs/contribute/style/style-guide/#patterns-to-avoid).

## Workflow

### Running Locally

To build the docs and start a server at <http://localhost:8000/> where you can preview your changes:

```bash
make docs-serve
```

This command also checks the docs for spelling, broken links, and lint issues.

The documentation of the ARC project is written primarily using Markdown.
All documentation related content can be found in <https://github.com/opendefensecloud/artifact-conduit/tree/main/docs>.
New content also should be added there.

To render the documentation with `mkdocs` locally you have to

- once or when you alter the [Dockerfile](https://github.com/opendefensecloud/artifact-conduit/blob/main/Dockerfile) you need to rebuild the custom container image with `make docs-docker-build`
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

### Entering a PR

On entering a PR, our CI will run the same checks as `make docs`, and fail the build if any issues are found.

## Tips

Use a service like [Grammarly](https://www.grammarly.com) to check your grammar.

Having your computer read text out loud is a way to catch problems, e.g.:

* Word substitutions (i.e. the wrong word is used, but spelled.
correctly).
* Sentences that do not read correctly will sound wrong.


