# Deployment

## Update Server

Run `update-server.exe` on the machine that can read `Z:\`.

```powershell
.\update-server.exe -config config\server.json
```

Firewall must allow inbound TCP `45000` from client PCs.

## Client Agent

Run `client-update.exe` on each client PC.

```powershell
.\client-update.exe -config config\agent.json
```

The client account must be allowed to write to `C:\tnl_appl`.
