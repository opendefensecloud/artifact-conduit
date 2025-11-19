FROM squidfunk/mkdocs-material

RUN pip install mkdocs-glightbox
RUN pip install mkdocs-include-markdown-plugin

CMD ["serve", "--dev-addr=0.0.0.0:8000", "--livereload"]