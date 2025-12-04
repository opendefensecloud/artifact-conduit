# Blob Example

How the file along with the checksum has been created and verified:

```bash
echo "hello world" > examples/blob/file.txt
sha256sum examples/blob/file.txt > file.txt.sha256
sha256sum -c examples/blob/file.txt.sha256
```
