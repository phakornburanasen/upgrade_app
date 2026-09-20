# Security

The server never accepts raw filesystem paths from clients. Clients provide only an application name and relative file paths issued by the server.

Rejected inputs include:

- `../`
- `..\`
- `C:\Windows`
- `\\server\share`
- absolute paths
- paths containing drive letters

The client stages files under `C:\tnl_appl\_update` before overwrite and backs up existing application folders under `C:\tnl_appl\_backup`.

HTTPS and token authentication are intentionally left as the next hardening step after v1 file transfer is verified.
