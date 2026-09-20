# Troubleshooting

## Server shows no apps

- Confirm `source_path` points to the correct folder.
- Confirm the server process account can read `Z:\`.
- Confirm each application is an immediate child folder of `Z:\`.

## Agent cannot connect

- Confirm `server_url` in `config\agent.json`.
- Confirm the update server is listening on port `45000`.
- Confirm firewall rules allow the connection.

## Update fails

- Check the local UI progress message.
- Check `logs\update-server.log`.
- Confirm the client can write to `C:\tnl_appl`.
- Confirm the selected application folder exists on the server.
