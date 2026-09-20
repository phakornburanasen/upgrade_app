# Lean Central Software Updater

This project provides a central Go update server and a Windows client update agent.

- Update server listens on `0.0.0.0:45000` and reads releases from `Z:\`.
- Client agent opens a local GUI-style browser UI on `http://127.0.0.1:45100`.
- Client updates write into `C:\tnl_appl\<ApplicationName>`.
- The client never reads `Z:` and never uses SMB for update transfer.
- v1 has no database, no version check, and no ZIP package step.

## Build

```powershell
go test ./...
go build -o bin\update-server.exe .\cmd\update-server
go build -ldflags="-H windowsgui" -o bin\client-update.exe .\cmd\update-agent
```

## Run

```powershell
.\bin\update-server.exe -config config\server.json
.\bin\client-update.exe -config config\agent.json
```

Open the client UI:

```text
http://127.0.0.1:45100
```
