# API

## Server

- `GET /api/health`
- `GET /api/apps`
- `GET /api/apps/{app}/files`
- `GET /api/apps/{app}/files/{relative_path}`
- `POST /api/update/report`
- `GET /api/update/logs`

## Agent

- `GET /`
- `GET /api/local/status`
- `GET /api/apps`
- `POST /api/update`
- `GET /api/update/status/{job_id}`

`POST /api/update` body:

```json
{
  "applications": ["Agent_TNLX", "SSO_Check"]
}
```
