# Install

1. Build both executables.
2. Put `update-server.exe` on the update server.
3. Ensure the update server account can read `Z:\`.
4. Put `client-update.exe` and `config\agent.json` on each Windows client.
5. Set `server_url` in `config\agent.json` to the update server, for example `http://update-server:45000`.
6. Run `client-update.exe`; it opens the local update UI automatically.
