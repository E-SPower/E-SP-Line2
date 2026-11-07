# Message fields

Every field of inbound messages, outbound commands and pushed events: meaning, type, allowed values and where it comes from.

- Defined in `internal/messagefields`; `GET /api/v1/message-fields` serves the same contract and `fields_test.go` checks the docs against it.
- Migrated from [catbus](https://github.com/cv-cat/catbus) `src/core/schemas.ts` and `AGENTS.md` 6.2 "normalized types". See the [catbus migration notes](./catbus-migration.md).

## Common rules

| Rule | Detail |
|---|---|
| Fields always exist | When a value is unavailable a nullable field is `null` and an array is `[]`. Never test for field absence. |
| IDs are always strings | 64-bit IDs (`aweme_id`, Kuaishou photo ids) are kept as strings so no precision is lost. **Do not** cast them to numbers. |
| Times are ISO 8601 with an offset | e.g. `2026-11-06T17:23:50+08:00`. Millisecond timestamps are converted in the platform's time zone. |
| Counts are integers or `null` | Display values such as `1.2万` are converted to `12000` first. |
| `id` and `url` are always present | An emitted object can be passed straight to the next command. `url` is `null` when the platform has no web page for the object. |
| Enums are closed | Only the values listed here are emitted; adding one means changing `messagefields`, this page and the tests together. |
| `--raw` swaps the object | Normalization drops platform-private fields. `--raw` replaces each normalized object with the platform's raw object; that shape varies by platform and is not stable. Scripts should prefer normalized fields. |

## Envelope

catbus writes exactly one envelope to stdout per call, identical in shape for success and failure. E-SP-Line2's HTTP API uses its own `{data, total}` / `{error}` responses; the envelope matters to adapter runtimes and to `hint`, which always names a `catbus …` command done in the adapter's own vocabulary.

```json
{
  "ok": false,
  "platform": "bilibili",
  "endpoint": "web",
  "resource": "item",
  "action": "get",
  "account": "default",
  "data": null,
  "page": null,
  "error": {
    "code": "AUTH_EXPIRED",
    "message": "credential expired",
    "hint": "catbus bilibili auth login",
    "detail": null
  }
}
```

| Field | Type | Meaning |
|---|---|---|
| `ok` | `boolean` | Whether the call succeeded. |
| `platform` | `string\|null` | Canonical platform id; aliases (`bili`, `tb`) are normalized first. `null` on global commands (`platforms`, `doctor`, `auth list`, `config`, `version`). |
| `endpoint` | `string\|null` | Endpoint: `web`, `app` or `pc`. `null` on global commands. |
| `resource` | `string\|null` | Resource (`item`, `user`, `msg`, …). On global commands it is the command name. |
| `action` | `string\|null` | Action (`get`, `search`, `send`, …). `null` when there is no subcommand. |
| `account` | `string\|null` | Account name used. |
| `data` | `object\|array\|null` | Result. `null` on failure. |
| `page` | `object\|null` | `{cursor, has_more}` on paged commands, otherwise `null`. |
| `error` | `object\|null` | `{code, message, hint, detail}` on failure, otherwise `null`. |

`error.code`, its exit code and what to do about it:

| Exit | `code` | Meaning | Next step |
|---|---|---|---|
| 1 | `ERROR` | Unclassified error | Read `message`; rerun with `-v` for debug logs |
| 2 | `USAGE` | Bad argument or option | Read `hint` and `--help` |
| 2 | `UNSUPPORTED` | Platform or endpoint lacks the command, or the value | `hint` names the canonical word, the choices, or `-e <endpoint>` |
| 2 | `CONFIRM_REQUIRED` | Dangerous operation not confirmed | Add `-y` when non-interactive |
| 3 | `AUTH_REQUIRED` | Not logged in, unknown account, or a login wall | Run the login command in `hint` |
| 3 | `AUTH_EXPIRED` | Credential expired | Log in again |
| 4 | `NOT_IMPLEMENTED` | Planned capability or endpoint | Unsupported for now, no point retrying |
| 5 | `RISK_CONTROL` | Captcha or risk control; `detail.kind` is `captcha`, `rate_limit` or `blocked` | Pause, do not retry in a tight loop |
| 6 | `NETWORK` | Network or proxy error | Check network and proxy |
| 7 | `UPSTREAM` | Platform returned a business error | Original code is in `detail` |

Resolution order: arguments and options first (`USAGE` for typos), then planned endpoints or capabilities (`NOT_IMPLEMENTED`), then concepts the platform lacks or commands outside the vocabulary (`UNSUPPORTED`).

`error.hint` always names the next command to run and can be piped straight into a shell. Using a platform's own word (`note`, `video`, `aweme`) produces a hint with the canonical one:

```json
{"ok":false,"resource":"note","data":null,"error":{"code":"UNSUPPORTED","message":"xhs (web) does not support note get","hint":"canonical word: catbus xhs item get","detail":null}}
```

## Message

`Message` carries direct-message bodies. `msg history`, `msg listen` and `msg send` all emit it.

| Field | Type | Empty | Meaning |
|---|---|---|---|
| `id` | `string` | required | Message id. |
| `conversation_id` | `string` | required | Conversation id, taken from the enclosing conversation. |
| `from` | `UserRef\|null` | nullable | Sender. `null` for system messages. |
| `type` | `string` | required | `text`, `image`, `video`, `card` or `other`. |
| `text` | `string\|null` | nullable | Text body. |
| `media` | `Media[]` | required | Images, video, audio. `[]` for text. |
| `created_at` | `string\|null` | nullable | Send time. `null` when the platform does not give it. |

### `type` values

| Value | When | `text` | `media` |
|---|---|---|---|
| `text` | the platform's content type is text | body | `[]` |
| `image` | the platform's content type is an image | `null` | the images, as many as given |
| `video` | a Douyin video message | `null` | the video |
| `card` | trade card, product card, share card, file card | the card summary; Douyin file messages put the file name here | `[]` |
| `other` | unrecognized content | whatever the platform gave | `[]` |

### Message shapes per platform

| Platform | Support | Shape source | Always null / notes |
|---|---|---|---|
| Xiaohongshu | full | `contentType`: 1 text, 2 image, 3 card, 4 video, else `other` | In groups `from` is `null` when the sender is missing; direct chats fall back to `is_self` to pick between self and peer. |
| Douyin | full | push and pull share one field set | File messages are `type: other` with the file name in `text`; share cards are `type: card`. |
| TikTok | full | text messages only | — |
| Xianyu | history and send only | `contentType`: 1 text, 2 image; trade cards become `card` | A card's `text` is the summary; the body is in `--raw`. |
| Taobao | history and send only | `contentType`: 1 text; 101 with an image is `image`, any other 101 is `card` | Chinese nicknames keep the raw cookie `_nk_` value, undecoded. |
| JD | full | Dongdong message `body.type` | The id field name of recalled messages is inferred. |
| X | `msg list` / `msg history` only | end-to-end encrypted | `msg history` yields placeholder messages only (`type: other`); the ciphertext is in `--raw`. |
| M-Team | `msg list` / `msg history` only | mailbox body | Read-only; needs a password session, API-key sessions get `UNSUPPORTED`. |

## Conversation

`msg list` emits `Conversation`.

| Field | Type | Empty | Meaning |
|---|---|---|---|
| `id` | `string` | required | Conversation id. Opaque, pass it back verbatim. |
| `peer` | `UserRef\|null` | nullable | The other side. `null` for groups and system notices. |
| `unread` | `integer\|null` | nullable | Unread count. |
| `last_message` | `string\|null` | nullable | Preview of the last message. |
| `updated_at` | `string\|null` | nullable | Last activity. |

Conversation id shape per platform:

| Platform | Shape | Notes |
|---|---|---|
| Xiaohongshu | the peer's user id; groups use `group:<group id>` | The group name is only in `--raw`. |
| Douyin | `0:1:<uid>:<uid>` | Inbox only; `msg list` is planned. |
| TikTok | platform conversation id | — |
| Xianyu | digits, possibly with an `@goofish` suffix | Comes from `msg listen` / `msg send` `conversation_id`. |
| Taobao | `<my id>.1-<peer id>.1#11001`, possibly with `@cntaobao` | Must belong to the current account. |
| JD | the shop's `venderId` | Self-operated support is `1`. |
| X | platform conversation id | Inbox first page only (last 20 conversations), no paging. |
| M-Team | mailbox id | `--box` picks `inbox`, `system` or `sent`; system messages have `peer` `{id: "0", name: "系统"}`. |

## User reference

### UserRef

`UserRef` is the shared shape everywhere a person is referenced.

| Field | Type | Empty | Meaning |
|---|---|---|---|
| `id` | `string` | required | User id, as a string. |
| `name` | `string\|null` | nullable | Nickname. `null` when the API does not return one. |
| `url` | `string\|null` | nullable | Profile link. `null` when the platform has no profiles. |

`User` adds `handle`, `avatar`, `bio` and `stats`:

| Field | Type | Empty | Meaning |
|---|---|---|---|
| `handle` | `string\|null` | nullable | Platform-unique short name (Douyin `unique_id`, X `screen_name`). |
| `avatar` | `string\|null` | nullable | Avatar. |
| `bio` | `string\|null` | nullable | Bio. |
| `stats.followers` / `stats.following` / `stats.items` / `stats.likes` | `integer\|null` | nullable | Followers, following, items, likes. |

`stats.*` is often permanently `null`: the Bilibili upstream user API has no counters and Kuaishou has no stats fields. Douyin item lists only carry `share_url`, so `author` only gets `id` and `name`.

## Notice

`notice list` emits `Notice`, `notice count` emits `NoticeCount`.

| `Notice` field | Type | Empty | Meaning |
|---|---|---|---|
| `id` | `string` | required | Notice id. |
| `type` | `string` | required | `comment`, `mention`, `like`, `follow` or `system`. |
| `user` | `UserRef\|null` | nullable | Who triggered it. `null` for system notices. |
| `target` | `object\|null` | nullable | The object acted on. `null` when detached. |
| `target.id` | `string` | required | Object id. |
| `target.url` | `string\|null` | nullable | Object link. |
| `text` | `string\|null` | nullable | Body. Template notices join title and body. |
| `created_at` | `string\|null` | nullable | When it happened. |

### NoticeCount (`notice_count`)

| Field | Type | Empty | Meaning |
|---|---|---|---|
| `total` | `integer` | required | Unread total. This is the platform's only number when the breakdown is all `null`. |
| `comment` / `mention` / `like` / `follow` / `system` | `integer\|null` | nullable | Unread per category. |

- Xiaohongshu: the `mentions` group splits into `comment` / `mention`, `likes` is `like`, `connections` is `follow`.
- TikTok: `group 500` is activity notices (split into follow / digg / comment / at), `group 661` is system notices.
- Kuaishou: only `unReadCount` is available, so even `total` is missing and the whole breakdown is `null`.
- YouTube: unread total only.
- Douyin: `notice list` only, `notice count` is planned.

## Live event

`live listen` emits one `Event` per line.

| Field | Type | Empty | Meaning |
|---|---|---|---|
| `type` | `string` | required | `chat`, `gift`, `like`, `enter`, `follow` or `other`. |
| `time` | `string` | required | Event time; the local receive time when the platform gives none. |
| `user` | `UserRef\|null` | nullable | Who triggered it. `null` for system messages. |
| `text` | `string\|null` | nullable | The chat line; for `like` events, the like count. |
| `gift` | `object\|null` | nullable | Gift. `null` unless `type` is `gift`. |
| `gift.name` | `string` | required | Gift name. |
| `gift.count` | `integer` | required | Combo / quantity. |

## Item

`Item` covers notes, works, videos, posts, goods and torrents, distinguished by `kind`.

| Field | Type | Empty | Meaning |
|---|---|---|---|
| `id` | `string` | required | Content id (BV number, `aweme_id`, `note_id`). |
| `kind` | `string` | required | `video`, `image`, `text`, `article`, `goods` or `torrent`. |
| `url` | `string\|null` | nullable | Detail page. |
| `title` | `string\|null` | nullable | Title. Douyin has only a description in `text`, so this is always `null` there. |
| `text` | `string\|null` | nullable | Body text. |
| `author` | `UserRef\|null` | nullable | Author. `null` for YouTube Shorts in lists. |
| `created_at` | `string\|null` | nullable | Publish time. |
| `cover` | `string\|null` | nullable | Cover; the first image. |
| `media` | `Media[]` | required | Media. |
| `stats` | `object` | required | Container. |
| `stats.views` | `integer\|null` | nullable | Views / plays. |
| `stats.likes` | `integer\|null` | nullable | Likes. |
| `stats.comments` | `integer\|null` | nullable | Comments. |
| `stats.collects` | `integer\|null` | nullable | Collects; the Xianyu "wants" count. |
| `stats.shares` | `integer\|null` | nullable | Shares / reposts. |
| `price` | `Price\|null` | nullable | `{amount, currency}`; `amount` is in the major unit, not cents. |
| `status` | `string\|null` | nullable | `published`, `reviewing`, `rejected`, `private`, `draft`, `on_sale`, `sold` or `off_shelf`. Only for own content. |

Permanently `null` examples: Douyin `title` and `stats.views` in Douyin and Xiaohongshu public lists. The platform notes carry the full list.

## Media

| Field | Type | Empty | Meaning |
|---|---|---|---|
| `id` | `string\|null` | nullable | Platform id. |
| `type` | `string` | required | `image`, `video` or `audio`. |
| `url` | `string\|null` | nullable | Reachable URL. `null` right after upload while transcoding. |
| `width` / `height` | `integer\|null` | nullable | Pixel size. |
| `duration` | `integer\|null` | nullable | Seconds. |

Multiple images keep platform order; the order is meaningful.

## Comment

| Field | Type | Empty | Meaning |
|---|---|---|---|
| `id` | `string` | required | Comment id. |
| `item_id` | `string` | required | Owning content id. |
| `parent_id` | `string\|null` | nullable | Comment replied to. Bilibili nested replies may point at another reply. |
| `author` | `UserRef\|null` | nullable | Author. |
| `text` | `string` | required | Body. |
| `created_at` | `string\|null` | nullable | Posted at. |
| `stats.likes` / `stats.replies` | `integer\|null` | nullable | Likes, replies. |

## Live

| Field | Type | Empty | Meaning |
|---|---|---|---|
| `id` | `string` | required | Room id. |
| `url` | `string\|null` | nullable | Room link. |
| `title` | `string\|null` | nullable | Title. |
| `status` | `string` | required | `live` or `offline`. |
| `host` | `UserRef\|null` | nullable | Host. |
| `cover` | `string\|null` | nullable | Cover. |
| `stats.viewers` | `integer\|null` | nullable | Concurrent viewers. |

## Inbound message

Platform pushes land in `InboundMessage`, filled by the adapter. Its fields mirror the objects above:

| Field | Type | Meaning |
|---|---|---|
| `platform_id` | `string` | Canonical platform id, matching the `platforms` table. |
| `instance_id` | `string` | Adapter instance id. |
| `conversation_id` | `string` | Conversation id, same shape as `Conversation.id`. |
| `sender_id` / `sender_name` | `string` | Sender, mirroring `Message.from` `id` / `name`. `sender_name` is an empty string when unavailable. |
| `message_type` | `string` | Mirrors `Message.type`. |
| `message_content` | `string` | The body as plain text; a summary when there are images or cards. |
| `raw_message` | `object` | Platform object, equivalent to `--raw`. |
| `idempotency_key` | `string` | `<platform>-<instance>-<message id>`. Repeated pushes dedupe on it. |
| `trace_id` | `string` | Trace id. |
| `timestamp` | `integer` | Milliseconds, the same instant as `created_at`. |
| `message_chain` | `object\|null` | Present only for multi-part messages, see below. |

### Message chain

A message made of several parts uses `MessageChain`; order is meaningful:

```json
{
  "id": "c1f0…",
  "timestamp": 1762419200000,
  "platform": "xianyu",
  "instance": "inst-1",
  "sender": {"id": "2200…", "name": "buyer"},
  "content": [
    {"type": "text", "content": {"text": "still available?"}},
    {"type": "product_card", "content": {"item_id": "891…", "title": "iPhone 15", "price": 3999, "detail_url": "https://…"}}
  ],
  "hash": "9f2c…"
}
```

`content[].type` and the element payload:

| `type` | `content` fields |
|---|---|
| `text` | `text` |
| `image` | `url`, `width`, `height`, `size`, `mime_type` |
| `audio` | `url`, `duration`, `size`, `mime_type` |
| `video` | `url`, `width`, `height`, `duration`, `size`, `mime_type`, `thumbnail` |
| `file` | `url`, `name`, `size`, `mime_type` |
| `product_card` | `item_id`, `title`, `price`, `image_url`, `detail_url`, `platform`, `sku`, `stock` |
| `order_info` | `order_id`, `status`, `amount`, `currency`, `item_count`, `created_at`, `paid_at`, `shipped_at`, `delivered_at` |
| `inquiry` | `product_id`, `order_id`, `question`, `category` (`price`, `shipping`, `quality`, `return`, `other`) |
| `location` | `latitude`, `longitude`, `address`, `name` |
| `emoji` | `emoji`, `name` |
| `mention` | `user_id`, `username` |

`order_info.status` is `pending`, `paid`, `shipped`, `delivered`, `cancelled` or `refunded`. Its time fields are milliseconds.

Limits: at most 100 elements, at most 1 MiB per message, `content[]` must not be empty, every element needs a `type`. `hash` is the MD5 of the chain with `raw` and `hash` itself removed; if it no longer recomputes, the chain was altered.

## Outbound command

`send_text`, `send_image`, `upload_media`, `create_conversation`, `get_history` and `refresh_token`.

| Field | Type | Meaning |
|---|---|---|
| `command_type` | `string` | One of the six above. |
| `instance_id` | `string` | Adapter instance that runs it. |
| `target_id` | `string` | Target conversation or user. Maps to `create_conversation` and to `msg send`'s `--to` / `--item`. |
| `payload` | `object` | Arguments. `send_text` is `{text}`, `send_image` is `{url, width, height}`. |
| `trace_id` | `string` | Trace id, matching the inbound message that triggered it. |
| `timestamp` | `integer` | Milliseconds. |

## Sending a direct message

| Form | Effect | Platforms |
|---|---|---|
| `msg send <text> --to <user>` | Open a conversation, then send | Xiaohongshu, Douyin, Xianyu |
| `msg send <text> --conversation <id>` | Send into an existing conversation | every platform that sends |
| `msg send <text> --item <item>` | Contact the item's seller | Xianyu, Taobao, JD |

- `--to` and `--conversation` are mutually exclusive; only `--to` with `--item` combine. Taobao has no `--to` because no API opens a conversation by user.
- Xianyu without `--item` opens the conversation on a hard-coded default item upstream; `--to` with `--item` contacts that user about that item, which lets a seller reach a buyer.
- On Douyin, giving `--image` together with `--video` makes the image the cover rather than a separate send.
- Audio, video and images go out in the order given; the last message is returned.
- Writes need a session; TikTok can only send into an existing conversation because the upstream cannot open one.

## Commands and their output types

| Command | Output type |
|---|---|
| `msg list` | `Conversation[]` |
| `msg history` | `Message[]` |
| `msg send` | `Message` |
| `msg listen` | `Message`, streamed |
| `notice list` | `Notice[]` |
| `notice count` | `NoticeCount` |
| `live listen` | `Event`, streamed |
| `live rank` | `Rank[]` |
| `live gifts` | `Gift[]` |
| `item get` / `item search` / `item list` | `Item` / `Item[]` |
| `comment list` / `comment replies` | `Comment[]` |
| `user get` / `user search` | `User` / `User[]` |
| `auth status` | `AuthStatus` |
| `item download` | `File[]` |

`Rank` is `{rank, user, score}`, `Gift` is `{id, name, price}`, `File` is `{path, type, url, size}`, `AuthStatus` is `{logged_in, user, method, expires_at}`.

## Paging

| Option | Meaning |
|---|---|
| (none) | One page |
| `--limit N` | Page automatically until N items |
| `--cursor <c>` | Continue from `page.cursor` |
| `--all` | Page to the end |

Cursors are opaque; pass them back unchanged. When `--limit` cuts a page, the cursor carries a `#skip=N` suffix and resuming re-fetches that page and skips the N already emitted. `has_more: false` means the end; if the server returns the same cursor twice, paging stops with a warning on stderr so `--all` cannot loop.

## Rank

`live rank` emits `Rank`.

| Field | Type | Empty | Meaning |
|---|---|---|---|
| `rank` | `integer` | required | Position, from 1. |
| `user` | `UserRef` | required | The ranked user. |
| `score` | `number\|null` | nullable | Score. `null` when the platform only ranks. |

## Gift

`live gifts` emits `Gift`.

| Field | Type | Empty | Meaning |
|---|---|---|---|
| `id` | `string` | required | Gift id. |
| `name` | `string` | required | Gift name. |
| `price` | `Price\|null` | nullable | Unit price. |

## File

`item download` emits `File[]`.

| Field | Type | Empty | Meaning |
|---|---|---|---|
| `path` | `string` | required | Local path. |
| `type` | `string` | required | `image`, `video`, `audio` or `torrent`. |
| `url` | `string\|null` | nullable | Temporary download URL. M-Team torrent links carry the personal passkey and are omitted. |
| `size` | `integer` | required | Bytes. |

## Auth status (`auth_status`)

`auth status` emits `AuthStatus`.

| Field | Type | Empty | Meaning |
|---|---|---|---|
| `logged_in` | `boolean` | required | Whether the session is valid. |
| `user` | `UserRef\|null` | nullable | The signed-in account. `null` when signed out or when the platform exposes no account. |
| `method` | `string\|null` | nullable | How the session was created: `qrcode`, `sms`, `password`, `cookie`, `oauth`, `apikey` or `guest`. |
| `expires_at` | `string\|null` | nullable | Expiry. `null` when the platform does not say. |

A `false` `logged_in` is not an error — it is a successful probe that found no session. `expires_at` is currently `null` on every platform, because none of them report an expiry and the local credential store is the only clock.

## Account

`auth list` emits `Account`.

| Field | Type | Empty | Meaning |
|---|---|---|---|
| `platform` | `string` | required | Canonical platform id. |
| `endpoint` | `string` | required | `web`, `app` or `pc`. |
| `account` | `string` | required | Account name. |
| `current` | `boolean` | required | Whether this is the active account. |
| `user` | `UserRef\|null` | nullable | The account's user. |
| `method` | `string\|null` | nullable | Login method. |
| `updated_at` | `string\|null` | nullable | Last update. |

## User reference shapes

`user_ref` is the shape above without `handle`, `avatar`, `bio` and `stats`. Use it wherever a person is mentioned but the profile is not loaded — `Message.from`, `Notice.user`, `Event.user`, `Comment.author`, `Live.host`, `Rank.user` and `Conversation.peer`. Use `user` only for a loaded profile (`user get`, `user search`).

## Related

- [Protocol v3](./protocol-v3.md): envelope, message chain and signing in full.
- [Adapter development](./adapter-development.md): how adapters fill these fields.
- [Platform capability matrix](./platform-capabilities.md): per-platform support and limitations.
