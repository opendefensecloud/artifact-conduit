# Blob Example

How the file along with the checksum has been created and verified:

```bash
echo "hello world" > examples/blob/file.txt
md5sum examples/blob/file.txt > file.txt.md5
md5sum -c examples/blob/file.txt.md5
```
