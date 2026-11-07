# Adapter configuration

## Overview

An adapter is the core component that connects to e-commerce platforms and handles message receipt and sending. Each adapter instance corresponds to a specific account or shop.

## Supported adapters

Every platform in the capability matrix ships a built-in adapter. The full list, with each platform's login methods and per-command support, is in the [platform capability matrix](../developer-guide/platform-capabilities.md) and served by `GET /api/v1/platforms/spec`.

Adapters come in two stages:

- **Active** (12): Xiaohongshu, Douyin, TikTok, Bilibili, Kuaishou, Weibo, Xianyu, Taobao, JD, X, YouTube, 12306. These are registered when the server starts.
- **Upcoming** (3): Jimeng, Maoyan Shows, M-Team. Registered as `inactive`; the capability table is kept so the same UI can show them.

Every adapter is `runtime_type: node`. Credentials are always per-platform: platforms that log in with a cookie take `cookie`, Jimeng takes the `dreamina` CLI credential, M-Team takes an API key, and 12306 needs none.

Capabilities are derived from the matrix, so an adapter only advertises what its platform actually supports. Alongside the plain capability words (`receive_message`, `send_text`) each supported command is listed as `<resource>.<action>`; planned commands carry an extra `<resource>.<action>.planned` marker.

## Create an adapter instance

### Step 1: Select an adapter package

On the Adapters page, click "Create instance" and select the adapter package you need.

### Step 2: Fill in the configuration

The config form is generated from the adapter's `config_schema`, so each platform only asks for what it needs. Common fields:

| Field | Type | Meaning |
|---|---|---|
| `cookie` | string | Platform login cookie. Required on every cookie platform. |
| `device_id` | string | Optional device identifier. |
| `heartbeat_interval` | integer | Heartbeat interval in seconds. Defaults to 15. |
| `reconnect_delay` | integer | Reconnect delay in seconds. Defaults to 5. |

Platform-specific credentials:

| Platform | Field | Notes |
|---|---|---|
| Jimeng | `token` | Login state written by `dreamina relogin`; macOS only. |
| M-Team | `api_key`, `uid` | API key sessions are read-only; `msg` needs a password session. `box` sets the default mailbox. |
| 12306 | — | No credentials. |

Xianyu example:

```json
{
  "cookie": "your_xianyu_cookie_here",
  "device_id": "optional_device_id",
  "heartbeat_interval": 15,
  "reconnect_delay": 5
}
```

Fields in `security.sensitive_fields` are encrypted at rest and masked in logs — every credential field is on that list (`heartbeat_interval` and `reconnect_delay` are not).

### Step 3: Configure routing rules

Configure routing rules for the adapter instance to determine how messages are dispatched.

### Step 4: Start the instance

Click the "Start" button; the system will:
1. Validate the configuration
2. Connect to the platform
3. Begin receiving messages

## Get a Cookie

Copy the cookie from an already signed-in browser:

1. Sign in to the platform's web site in a normal window.
2. Press F12 to open the developer tools.
3. Switch to the "Network" tab and refresh the page.
4. Find any same-site request and copy the entire `Cookie` request header.
5. Paste it into the adapter's `cookie` field.

Export as the site you are actually logged in to: Xiaohongshu from `xiaohongshu.com`, Xianyu from `goofish.com`, and so on. A cookie from the wrong host is rejected at connect time.

Some platforms need more than a cookie — Douyin and TikTok write operations also need the signer material produced by the same sign-in (see the platform's notes in the capability matrix). YouTube needs `SAPISID` in the cookie, which is HttpOnly, so copy it from the request header rather than `document.cookie`.

Logging in through the browser that already holds the session and then closing that window is best: Google and some other platforms rotate session cookies while the browser keeps using them, and the exported copy then expires within minutes.

## Adapter status

An adapter instance has the following statuses:

- **Stopped**: stopped
- **Starting**: starting
- **Running**: running
- **Stopping**: stopping
- **Error**: error state
- **Disconnected**: disconnected

## Monitor an adapter

### Connection status

On the Adapters page you can view:
- Current status
- Connection time
- Last heartbeat time
- Message count
- Error count

### View logs

Click the "Logs" button of an adapter instance to view:
- Connection logs
- Message send and receive logs
- Error logs

## Adapter management

### Start/stop

- **Start**: establish a connection to the platform and begin receiving messages
- **Stop**: disconnect and stop receiving messages

### Restart

Restarting an adapter will:
1. Stop the current connection
2. Validate the configuration again
3. Establish a new connection

### Delete

Deleting an adapter instance will:
1. Stop the instance (if it is running)
2. Delete the configuration data
3. Delete the related routing rules

## Best practices

### Cookie management

1. **Update regularly**: Cookies expire, so update them regularly
2. **Store securely**: Cookies contain sensitive information, so keep storage secure
3. **Monitor status**: watch the adapter status to catch an invalid Cookie early

### Performance tuning

1. **Heartbeat interval**: set a suitable heartbeat interval according to platform requirements
2. **Reconnect strategy**: configure a sensible reconnect delay and maximum retry count
3. **Queue size**: adjust the queue size to the message volume

### Failure handling

1. **Automatic reconnect**: the system retries the connection automatically
2. **Error alerts**: configure error alerts to spot problems early
3. **Log analysis**: analyze the logs regularly and tune the configuration

## Troubleshooting

### Invalid Cookie

**Symptoms**: the adapter status changes to Error and the logs show an authentication failure

**Resolution**:
1. Get the Cookie again
2. Update the adapter configuration
3. Restart the adapter

### Connection timeout

**Symptoms**: the adapter status stays Starting and does not change to Running for a long time

**Resolution**:
1. Check the network connection
2. Check the platform service status
3. Increase the timeout setting

### Message loss

**Symptoms**: some messages are not received

**Resolution**:
1. Check the routing rule configuration
2. Check the message tracking logs
3. Confirm that the downstream application is online

## Advanced configuration

### Custom heartbeat interval

```json
{
  "heartbeat_interval": 30
}
```

### Custom reconnect strategy

```json
{
  "reconnect_delay": 10,
  "max_retries": 5
}
```

### Proxy configuration

```json
{
  "proxy": {
    "host": "proxy.example.com",
    "port": 8080,
    "username": "user",
    "password": "pass"
  }
}
```
