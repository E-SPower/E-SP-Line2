# Platform capability matrix

Per-platform web capability support. Migrated from catbus `docs/capabilities.md` (MIT); see the [catbus migration notes](./catbus-migration.md).

- Generated from the capability matrix by `scripts/gen_capabilities.py`; **do not hand-edit**. A test checks that it is current.
- The same matrix feeds `internal/platformspec`; the command tree, adapter manifests and `GET /api/v1/platforms/spec` all derive from it.
- The app and pc endpoints are all planned and are not listed.

| Symbol | Meaning | Support | Runtime result |
|---|---|---|---|
| ✓ | upstream has it | `implemented` | runs |
| ◐ | upstream has it in part; the limitation is in the platform notes | `partial` | runs |
| ○ | the platform has the concept, upstream does not; planned | `planned` | `NOT_IMPLEMENTED`, exit code 4 |
| — | the platform has no such concept | `unsupported` | `UNSUPPORTED`, exit code 2 |

Platforms come in two stages: **active** (xhs · douyin · tiktok · bilibili · kuaishou · weibo · xianyu · taobao · jd · x · youtube) and **upcoming** (12306 · jimeng · maoyan · mteam). Upcoming platforms register as `inactive` adapters while keeping their rows.

## Active

### auth

| 命令 | xhs | douyin | tiktok | bilibili | kuaishou | weibo | xianyu | taobao | jd | x | youtube |
|---|---|---|---|---|---|---|---|---|---|---|---|
| `auth status` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ◐ | ✓ | ✓ | ✓ |

- taobao `auth status`: the upstream has no current-user API: it spends one DM token to verify the session, and takes the account from the cookie's unb and nickname

### user

| 命令 | xhs | douyin | tiktok | bilibili | kuaishou | weibo | xianyu | taobao | jd | x | youtube |
|---|---|---|---|---|---|---|---|---|---|---|---|
| `user get` | ✓ | ✓ | ◐ | ✓ | ✓ | ✓ | ◐ | ◐ | ◐ | ✓ | ✓ |
| `user search` | ✓ | ✓ | ○ | ✓ | ✓ | ○ | — | — | — | ✓ | ✓ |
| `user items` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ○ | ○ | — | ✓ | ✓ |
| `user likes` | ✓ | ✓ | ○ | — | ✓ | ○ | — | — | — | ○ | ✓ |
| `user collects` | ✓ | ○ | ✓ | ○ | ✓ | ○ | ○ | ○ | ✓ | ○ | ✓ |
| `user reposts` | — | — | ✓ | — | — | ○ | — | — | — | ○ | — |
| `user followers` | ○ | ✓ | ✓ | ○ | ✓ | ○ | — | — | — | ○ | — |
| `user following` | ◐ | ✓ | ✓ | ○ | ✓ | ○ | — | — | — | ○ | ✓ |
| `user follow` | ○ | ○ | ✓ | ○ | ○ | ○ | ○ | — | — | ✓ | ✓ |
| `user unfollow` | ○ | ○ | ✓ | ○ | ○ | ○ | ○ | — | — | ✓ | ✓ |
| `user update` | — | — | — | ◐ | — | — | — | — | — | — | — |

- tiktok `user get`: me is partially supported
- xianyu `user get`: me only; querying others is planned
- taobao `user get`: <item> only accepts product links (Taobao / Tmall product page or an m.tb.cn short link) and returns that product's seller; me is planned
- jd `user get`: me only
- youtube `user get`: <user> is a channel id (UC…), an @handle or a channel link
- youtube `user items`: the channel's Videos tab, without Shorts or live
- kuaishou `user likes`: me only
- youtube `user likes`: own only (me), i.e. the Liked videos playlist
- youtube `user collects`: own only (me), i.e. the Watch Later list
- kuaishou `user followers`: me only
- xhs `user following`: me only
- kuaishou `user following`: me only
- youtube `user following`: own subscriptions only (me)
- youtube `user follow`: subscribe to the channel
- youtube `user unfollow`: unsubscribe from the channel
- bilibili `user update`: not upstream: upstream has no profile-edit API, so the request follows the account.bilibili.com profile page and has no fixture comparison. Renaming costs 6 coins each time (free once for an account that never set a nickname) and needs confirming; the signature is at most 70 characters and is reviewed after the change; avatars accept png / jpg only, at most 2 MB

### item

| 命令 | xhs | douyin | tiktok | bilibili | kuaishou | weibo | xianyu | taobao | jd | x | youtube |
|---|---|---|---|---|---|---|---|---|---|---|---|
| `item get` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ○ | ✓ | ✓ | ✓ |
| `item search` | ✓ | ✓ | ✓ | ✓ | ◐ | ◐ | ○ | ○ | ✓ | ✓ | ✓ |
| `item related` | ○ | ○ | ✓ | ✓ | ✓ | — | ○ | ○ | ◐ | — | ✓ |
| `item list` | ✓ | ◐ | ✓ | ✓ | ✓ | ○ | ○ | — | — | — | ✓ |
| `item media` | ✓ | ✓ | ◐ | ✓ | ✓ | ○ | — | — | — | ✓ | ◐ |
| `item download` | ✓ | ✓ | ◐ | ✓ | ✓ | ○ | — | — | — | ✓ | ◐ |
| `item like` | ○ | ✓ | ✓ | ✓ | ○ | ○ | — | — | — | ✓ | ✓ |
| `item unlike` | ○ | ✓ | ✓ | ✓ | ○ | ○ | — | — | — | ✓ | ✓ |
| `item collect` | ○ | ✓ | ✓ | ✓ | ○ | ○ | ○ | ○ | ○ | ✓ | ✓ |
| `item uncollect` | ○ | ✓ | ✓ | ✓ | ○ | ○ | ○ | ○ | ○ | ✓ | ✓ |
| `item repost` | — | — | ○ | — | — | ○ | — | — | — | ✓ | — |
| `item unrepost` | — | — | ○ | — | — | ○ | — | — | — | ✓ | — |
| `item publish` | ✓ | ✓ | ✓ | ✓ | ◐ | ✓ | ✓ | — | — | ✓ | ◐ |
| `item delete` | ○ | ○ | ○ | ◐ | ○ | ○ | ○ | — | — | ✓ | ✓ |
| `item categories` | — | — | — | ✓ | — | — | ○ | — | — | — | — |

- douyin `item search`: --type video uses the video channel search, anything else uses the general channel; --type image returns image posts only
- kuaishou `item search`: no filters (--sort / --type / --time); keyword only
- weibo `item search`: first page only (m.weibo.cn's guest asks for a session from page two on)
- x `item search`: --type video / image uses media search and filters by type; cannot combine with --sort latest
- jd `item related`: uses the results of the first related search term
- douyin `item list`: returns the works preview from the publish page (work_list); a scheduled unpublished work has status draft
- youtube `item list`: videos uploaded to your own channel, from YouTube Studio
- tiktok `item media`: the media URL comes from the video detail (video.playAddr; imagePost for image posts); upstream has no separate resolve API
- youtube `item media`: best video track plus audio track, split. Uses the VISIONOS client without a session; too many requests from one egress IP trigger a bot check; age-restricted and similar videos get no URL
- tiktok `item download`: the media URL is the same as item media
- youtube `item download`: downloads the item media video track and audio track as two files, without muxing
- youtube `item unlike`: unlike (also clears a dislike)
- youtube `item collect`: add to Watch Later
- youtube `item uncollect`: remove from Watch Later
- kuaishou `item publish`: no --cover, --mention, --poi, --category or --price; --image takes 1–31 images (15 MB each) and excludes --video
- x `item publish`: bodies over 280 weight post as a long post (needs Premium); --thread posts a thread
- youtube `item publish`: goes through YouTube Studio: chunked upload, and creating a video requires a BotGuard check, so local Chromium is needed (see AGENTS 5.7). The account needs a channel first; --schedule creates it private and flips it public at the time. Video upload is verified on a real machine. --cover requires the channel's custom-thumbnail feature (phone verification) and errors before upload when it is off
- bilibili `item delete`: needs a captcha (Geetest click-to-select) that catbus cannot pass automatically; reports RISK_CONTROL
- youtube `item delete`: own videos only; also needs local Chromium for the BotGuard check

### product and reviews

| 命令 | xhs | douyin | tiktok | bilibili | kuaishou | weibo | xianyu | taobao | jd | x | youtube |
|---|---|---|---|---|---|---|---|---|---|---|---|
| `product get` | ○ | ◐ | ✓ | — | ○ | — | — | — | — | — | — |

- douyin `product get`: the upstream product API returns detail images, specs and a jump link only: title, image and price, with the other fields empty

### comment

| 命令 | xhs | douyin | tiktok | bilibili | kuaishou | weibo | xianyu | taobao | jd | x | youtube |
|---|---|---|---|---|---|---|---|---|---|---|---|
| `comment list` | ◐ | ✓ | ✓ | ◐ | ◐ | ◐ | — | ○ | ◐ | ✓ | ✓ |
| `comment replies` | ✓ | ✓ | ✓ | ◐ | ◐ | ○ | — | — | — | ○ | ✓ |
| `comment add` | ○ | ✓ | ✓ | ◐ | ○ | ○ | — | — | — | ✓ | ✓ |
| `comment delete` | ○ | ○ | ○ | ◐ | ○ | ○ | — | — | — | ✓ | ✓ |
| `comment like` | ○ | ○ | ○ | ○ | ○ | ○ | — | — | — | ✓ | ✓ |
| `comment unlike` | ○ | ○ | ○ | ○ | ○ | ○ | — | — | — | ✓ | ✓ |

- xhs `comment list`: --product is planned
- bilibili `comment list`: supports videos, articles, text / repost dynamics and image dynamics; image dynamics get the album comment-section id from a non-upstream dynamic detail API, with no fixture comparison
- kuaishou `comment list`: --product is planned
- weibo `comment list`: first-level comments only
- jd `comment list`: first page only; --limit N takes N comments in one request
- bilibili `comment replies`: not upstream: the nested-reply and dynamic-detail APIs, with no fixture comparison; covers visible replies under video, article and image-dynamic root comments; nested replies for text and repost dynamics are not verified and unsupported
- kuaishou `comment replies`: the upstream nested-comment API was checked against one captured parameter set only; results for other comments were not checked upstream
- youtube `comment replies`: <comment> is a root comment id; YouTube replies are one level deep
- bilibili `comment add`: dynamics support text and repost only: image dynamics (with images) hang their comment section on an album and upstream has no album-id API; videos and articles are unaffected
- youtube `comment add`: --reply-to takes a root comment or a reply; all replies hang under the root
- bilibili `comment delete`: dynamics support text and repost only: image dynamics (with images) hang their comment section on an album and upstream has no album-id API; videos and articles are unaffected
- youtube `comment delete`: own comments only

### feed and keyword

| 命令 | xhs | douyin | tiktok | bilibili | kuaishou | weibo | xianyu | taobao | jd | x | youtube |
|---|---|---|---|---|---|---|---|---|---|---|---|
| `feed list` | ◐ | ◐ | ✓ | ◐ | ◐ | ○ | ○ | ○ | ○ | ◐ | ✓ |
| `feed categories` | ✓ | — | ○ | — | — | — | — | — | — | — | — |
| `keyword suggest` | ✓ | ○ | ✓ | ○ | ○ | ○ | ○ | ○ | ✓ | ○ | ✓ |
| `keyword hot` | ✓ | ○ | ○ | ○ | ○ | ○ | — | — | ✓ | ○ | — |

- xhs `feed list`: following is planned
- douyin `feed list`: hot and following are planned
- bilibili `feed list`: following is not upstream: upstream has no feed API, so the request follows the web dynamic page (x/polymer/web-dynamic/v1/feed/all) with no fixture comparison; lists videos from people you follow and needs a session
- kuaishou `feed list`: following is first screen only
- x `feed list`: following is planned
- youtube `feed list`: home recommendations and the subscription feed; the guest home is empty, so a session is needed

### notice and msg

| 命令 | xhs | douyin | tiktok | bilibili | kuaishou | weibo | xianyu | taobao | jd | x | youtube |
|---|---|---|---|---|---|---|---|---|---|---|---|
| `notice list` | ✓ | ✓ | ✓ | ○ | ○ | ○ | — | — | — | ○ | ✓ |
| `notice count` | ✓ | ○ | ✓ | ○ | ✓ | ○ | — | — | — | ○ | ✓ |
| `msg list` | ✓ | ○ | ✓ | ○ | ○ | ○ | ○ | ○ | ✓ | ◐ | — |
| `msg history` | ✓ | ○ | ✓ | ○ | ○ | ○ | ✓ | ✓ | ✓ | ◐ | — |
| `msg send` | ✓ | ✓ | ◐ | ○ | ○ | ○ | ✓ | ✓ | ✓ | ○ | — |
| `msg listen` | ✓ | ✓ | ✓ | ○ | ○ | ○ | ✓ | ✓ | ✓ | ○ | — |
| `msg read` | ✓ | ○ | ○ | ○ | ○ | ○ | ○ | ○ | ○ | ○ | — |
| `msg revoke` | ✓ | ○ | ○ | ○ | ○ | ○ | ○ | ○ | ○ | ○ | — |
| `msg delete` | ✓ | ○ | ○ | ○ | ○ | ○ | ○ | ○ | ○ | ○ | — |

- youtube `notice count`: unread total only, no breakdown
- xhs `msg list`: includes groups; a group conversation id is group:<group id>
- x `msg list`: inbox first page only (last 20 conversations); upstream has no paging
- xhs `msg history`: direct chats take the peer id, groups take group:<group id>
- x `msg history`: messages are end-to-end encrypted, so only placeholder messages are emitted
- tiktok `msg send`: text only, and only to a user with an existing conversation (upstream cannot open one)
- xianyu `msg send`: without --item, --to opens the conversation on the upstream hard-coded default item; with --item, --to contacts that user about that item (a seller can reach a buyer)
- taobao `msg send`: --item contacts the product seller and --conversation replies into an existing conversation; --to is unsupported (there is no API to open a conversation by user)

### media, folder, series, history, topic, poi

| 命令 | xhs | douyin | tiktok | bilibili | kuaishou | weibo | xianyu | taobao | jd | x | youtube |
|---|---|---|---|---|---|---|---|---|---|---|---|
| `media upload` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | — | ✓ | — |
| `folder list` | ◐ | ✓ | ✓ | ◐ | ○ | ○ | — | — | — | ○ | — |
| `folder items` | ○ | ○ | ✓ | ◐ | ○ | ○ | — | — | — | ○ | — |
| `folder create` | ○ | ○ | ✓ | ○ | ○ | — | — | — | — | ○ | — |
| `folder update` | ○ | ○ | ✓ | ○ | ○ | — | — | — | — | ○ | — |
| `folder delete` | ○ | ○ | ○ | ○ | ○ | — | — | — | — | ○ | — |
| `series list` | ○ | ○ | ✓ | ○ | ○ | — | — | — | — | — | ✓ |
| `series items` | ○ | ○ | ○ | ○ | ○ | — | — | — | — | — | ✓ |
| `history list` | ○ | ○ | ○ | ○ | ○ | ○ | ○ | ○ | ✓ | — | ✓ |
| `topic search` | ✓ | ○ | ○ | ○ | ○ | ○ | — | — | — | — | — |
| `poi search` | ✓ | ○ | ◐ | — | ○ | ○ | — | — | — | — | — |

- xhs `folder list`: lists public collections only (upstream has that one API); upstream has no items-in-collection API
- bilibili `folder list`: lists only collections the user created, all in one response
- bilibili `folder items`: not upstream: upstream has no collection-content API, so the request follows the web collection page (x/v3/fav/resource/list) with no fixture comparison; videos only
- youtube `series list`: the channel's public playlists; me is all your playlists (needs a session)
- youtube `series items`: <series> is a playlist id (PL…) or a link carrying list=
- tiktok `poi search`: filters the recommended places by keyword

### live

| 命令 | xhs | douyin | tiktok | bilibili | kuaishou | weibo | xianyu | taobao | jd | x | youtube |
|---|---|---|---|---|---|---|---|---|---|---|---|
| `live get` | ✓ | ✓ | ✓ | ✓ | ✓ | ○ | — | ○ | — | — | ✓ |
| `live list` | ✓ | ○ | ✓ | ○ | ✓ | ○ | — | ○ | — | — | — |
| `live search` | ○ | ✓ | ✓ | ✓ | ○ | ○ | — | ○ | — | — | ✓ |
| `live categories` | ✓ | ○ | ✓ | ✓ | ✓ | ○ | — | ○ | — | — | — |
| `live listen` | ✓ | ✓ | ✓ | ✓ | ✓ | ○ | — | ○ | — | — | ✓ |
| `live history` | ○ | ◐ | ◐ | ✓ | ○ | ○ | — | ○ | — | — | ✓ |
| `live send` | ◐ | ◐ | ◐ | ✓ | ○ | ○ | — | ○ | — | — | ✓ |
| `live like` | ○ | ✓ | ✓ | ○ | ○ | ○ | — | ○ | — | — | — |
| `live rank` | ○ | ✓ | ✓ | ○ | ○ | ○ | — | ○ | — | — | — |
| `live gifts` | ✓ | ○ | ✓ | ✓ | ✓ | ○ | — | ○ | — | — | — |
| `live products` | ✓ | ✓ | ○ | — | ○ | ○ | — | ○ | — | — | — |
| `live media` | ○ | ◐ | ◐ | ✓ | ○ | ○ | — | ○ | — | — | — |
| `live replays` | — | ○ | — | ○ | ✓ | ○ | — | ○ | — | — | — |
| `live start` | ○ | ○ | ○ | ✓ | ○ | ○ | — | ○ | — | — | — |
| `live stop` | ○ | ○ | ○ | ✓ | ○ | ○ | — | ○ | — | — | — |

- youtube `live get`: <room> is the live video id or link
- tiktok `live list`: only followed people who are live (upstream get_webcast_feed is the follow list in the live page sidebar); empty when nobody followed is live
- youtube `live listen`: polls the live chat: text messages, Super Chat, memberships
- douyin `live history`: only the last 15 messages fetched when entering the room
- tiktok `live history`: only the danmaku fetched when entering the room (im/fetch)
- xhs `live send`: --gift is planned
- douyin `live send`: --gift is planned
- tiktok `live send`: --gift is planned
- youtube `live send`: text only; --gift is unsupported
- douyin `live media`: the stream URL comes from the room profile (room/web/enter)
- tiktok `live media`: upstream does not resolve the stream URL: take liveRoom.streamData from /api-live/user/room, or stream_url from room/enter

### 平台扩展

| 命令 | xhs | douyin | tiktok | bilibili | kuaishou | weibo | xianyu | taobao | jd | x | youtube |
|---|---|---|---|---|---|---|---|---|---|---|---|
| `folder add` | — | — | ✓ | — | — | — | — | — | — | — | — |
| `item edit` | — | — | — | ✓ | — | — | — | — | — | — | — |
| `item coin` | — | — | — | ✓ | — | — | — | — | — | — | — |
| `item triple` | — | — | — | ✓ | — | — | — | — | — | — | — |
| `item subtitles` | — | — | — | ✓ | — | — | — | — | — | — | — |
| `comment pin` | — | — | — | ✓ | — | — | — | — | — | — | — |
| `comment unpin` | — | — | — | ✓ | — | — | — | — | — | — | — |
| `danmaku list` | — | — | — | ✓ | — | — | — | — | — | — | — |
| `danmaku send` | — | — | — | ✓ | — | — | — | — | — | — | — |
| `dynamic publish` | — | — | — | ✓ | — | — | — | — | — | — | — |
| `dynamic delete` | — | — | — | ✓ | — | — | — | — | — | — | — |
| `article publish` | — | — | — | ✓ | — | — | — | — | — | ✓ | — |
| `draft get` | — | — | — | ✓ | — | — | — | — | — | — | — |
| `draft delete` | — | — | — | ✓ | — | — | — | — | — | — | — |
| `order list` | — | — | — | — | — | — | — | — | ✓ | — | — |
| `cart count` | — | — | — | — | — | — | — | — | ✓ | — | — |
| `coupon list` | — | — | — | — | — | — | — | — | ✓ | — | — |
| `article delete` | — | — | — | — | — | — | — | — | — | ✓ | — |
| `channel create` | — | — | — | — | — | — | — | — | — | — | ✓ |


## Upcoming

### auth

| 命令 | 12306 | jimeng | maoyan | mteam |
|---|---|---|---|---|
| `auth status` | — | ✓ | ✓ | ✓ |


### user

| 命令 | 12306 | jimeng | maoyan | mteam |
|---|---|---|---|---|
| `user get` | — | — | — | ◐ |
| `user collects` | — | — | — | ◐ |

- mteam `user get`: me only; upload, download and other account fields are available via --raw
- mteam `user collects`: me only; listed per section (--mode)

### item

| 命令 | 12306 | jimeng | maoyan | mteam |
|---|---|---|---|---|
| `item get` | — | — | ✓ | ✓ |
| `item search` | — | — | ○ | ✓ |
| `item download` | — | — | — | ✓ |
| `item collect` | — | — | — | ✓ |
| `item uncollect` | — | — | — | ✓ |
| `item categories` | — | — | — | ✓ |

- maoyan `item get`: show time, venue, price tier and sale state are written into text; --tickets needs a session
- mteam `item download`: the torrent file carries the personal passkey and is written 0600; the temporary download URL is not printed. --qbit pushes to the local qBittorrent Web UI (CATBUS_MTEAM_QBIT_URL, default http://127.0.0.1:25254) and --qbit-savepath sets the save directory

### feed and keyword

| 命令 | 12306 | jimeng | maoyan | mteam |
|---|---|---|---|---|
| `feed list` | — | — | ✓ | ✓ |
| `feed categories` | — | — | ✓ | — |

- maoyan `feed list`: returns every hot show in that city at once, unpaged; --category filters locally

### notice and msg

| 命令 | 12306 | jimeng | maoyan | mteam |
|---|---|---|---|---|
| `msg list` | — | — | — | ◐ |
| `msg history` | — | — | — | ◐ |

- mteam `msg list`: read-only and needs a password session; --box picks inbox / system / sent
- mteam `msg history`: <conversation> is an id from msg list; it is not marked read

### media, folder, series, history, topic, poi

| 命令 | 12306 | jimeng | maoyan | mteam |
|---|---|---|---|---|
| `poi search` | — | — | ◐ | — |

- maoyan `poi search`: searches show cities only (built-in city table); the id is the feed list --city value

### 平台扩展

| 命令 | 12306 | jimeng | maoyan | mteam |
|---|---|---|---|---|
| `station search` | ✓ | — | — | — |
| `ticket search` | ✓ | — | — | — |
| `ticket price` | ✓ | — | — | — |
| `route get` | ✓ | — | — | — |
| `transfer search` | ✓ | — | — | — |
| `seed list` | — | — | — | ✓ |
| `seed stats` | — | — | — | ✓ |
| `exam get` | — | — | — | ✓ |
| `bonus get` | — | — | — | ✓ |
| `bonus history` | — | — | — | ✓ |


## Platforms

- `xhs` — 笔记 (active) · [docs](https://github.com/cv-cat/Spider_XHS)
- `douyin` — 作品 (active) · [docs](https://github.com/cv-cat/DouYin_Spider)
- `tiktok` — 视频 (active) · [docs](https://github.com/cv-cat/TiktokApis)
- `bilibili` — 稿件 (active) · [docs](https://github.com/cv-cat/BilibiliApis)
- `kuaishou` — 作品 (active) · [docs](https://github.com/cv-cat/KuaiShou-Spider)
- `weibo` — 微博 (active) · [docs](https://github.com/cv-cat/WeiboApis)
- `xianyu` — 闲置商品 (active) · [docs](https://github.com/cv-cat/XianYuApis)
- `taobao` — 商品 (active) · [docs](https://github.com/cv-cat/TaoBaoApis)
- `jd` — 商品 SKU (active) · [docs](https://github.com/cv-cat/JdApis)
- `x` — 推文 (active) · [docs](https://github.com/cv-cat/XApis)
- `youtube` — 视频 (active)
- `12306` — 车次 (upcoming)
- `jimeng` — 作品 (upcoming)
- `maoyan` — 演出 (upcoming)
- `mteam` — 种子资源 (upcoming)

## Inline notes

- taobao `auth status`: the upstream has no current-user API: it spends one DM token to verify the session, and takes the account from the cookie's unb and nickname
- tiktok `user get`: me is partially supported
- xianyu `user get`: me only; querying others is planned
- taobao `user get`: <item> only accepts product links (Taobao / Tmall product page or an m.tb.cn short link) and returns that product's seller; me is planned
- jd `user get`: me only
- youtube `user get`: <user> is a channel id (UC…), an @handle or a channel link
- mteam `user get`: me only; upload, download and other account fields are available via --raw
- youtube `user items`: the channel's Videos tab, without Shorts or live
- kuaishou `user likes`: me only
- youtube `user likes`: own only (me), i.e. the Liked videos playlist
- youtube `user collects`: own only (me), i.e. the Watch Later list
- mteam `user collects`: me only; listed per section (--mode)
- kuaishou `user followers`: me only
- xhs `user following`: me only
- kuaishou `user following`: me only
- youtube `user following`: own subscriptions only (me)
- youtube `user follow`: subscribe to the channel
- youtube `user unfollow`: unsubscribe from the channel
- bilibili `user update`: not upstream: upstream has no profile-edit API, so the request follows the account.bilibili.com profile page and has no fixture comparison. Renaming costs 6 coins each time (free once for an account that never set a nickname) and needs confirming; the signature is at most 70 characters and is reviewed after the change; avatars accept png / jpg only, at most 2 MB
- all `user search --fans`: douyin
- all `user search --user-type`: douyin
- all `user items --keyword`: bilibili
- all `user collects --area`: jd
- all `user collects --mode`: mteam
- maoyan `item get`: show time, venue, price tier and sale state are written into text; --tickets needs a session
- douyin `item search`: --type video uses the video channel search, anything else uses the general channel; --type image returns image posts only
- kuaishou `item search`: no filters (--sort / --type / --time); keyword only
- weibo `item search`: first page only (m.weibo.cn's guest asks for a session from page two on)
- x `item search`: --type video / image uses media search and filters by type; cannot combine with --sort latest
- jd `item related`: uses the results of the first related search term
- douyin `item list`: returns the works preview from the publish page (work_list); a scheduled unpublished work has status draft
- youtube `item list`: videos uploaded to your own channel, from YouTube Studio
- tiktok `item media`: the media URL comes from the video detail (video.playAddr; imagePost for image posts); upstream has no separate resolve API
- youtube `item media`: best video track plus audio track, split. Uses the VISIONOS client without a session; too many requests from one egress IP trigger a bot check; age-restricted and similar videos get no URL
- tiktok `item download`: the media URL is the same as item media
- youtube `item download`: downloads the item media video track and audio track as two files, without muxing
- mteam `item download`: the torrent file carries the personal passkey and is written 0600; the temporary download URL is not printed. --qbit pushes to the local qBittorrent Web UI (CATBUS_MTEAM_QBIT_URL, default http://127.0.0.1:25254) and --qbit-savepath sets the save directory
- youtube `item unlike`: unlike (also clears a dislike)
- youtube `item collect`: add to Watch Later
- youtube `item uncollect`: remove from Watch Later
- kuaishou `item publish`: no --cover, --mention, --poi, --category or --price; --image takes 1–31 images (15 MB each) and excludes --video
- x `item publish`: bodies over 280 weight post as a long post (needs Premium); --thread posts a thread
- youtube `item publish`: goes through YouTube Studio: chunked upload, and creating a video requires a BotGuard check, so local Chromium is needed (see AGENTS 5.7). The account needs a channel first; --schedule creates it private and flips it public at the time. Video upload is verified on a real machine. --cover requires the channel's custom-thumbnail feature (phone verification) and errors before upload when it is off
- bilibili `item delete`: needs a captcha (Geetest click-to-select) that catbus cannot pass automatically; reports RISK_CONTROL
- youtube `item delete`: own videos only; also needs local Chromium for the BotGuard check
- all `item get --area`: jd
- all `item get --tickets`: maoyan
- all `item search --length`: douyin
- all `item search --range`: douyin
- all `item search --area`: jd
- all `item search --mode`: mteam
- all `item search --category`: mteam
- all `item search --free`: mteam
- all `item search --order-by`: mteam
- all `item search --asc`: mteam
- all `item related --area`: jd
- all `item download --qbit`: mteam
- all `item download --qbit-category`: mteam
- all `item download --qbit-paused`: mteam
- all `item download --qbit-savepath`: mteam
- all `item collect --folder`: douyin, bilibili
- all `item uncollect --folder`: douyin, bilibili
- all `item publish --poi-name`: xhs, douyin, weibo
- all `item publish --series`: douyin
- all `item publish --hotspot`: douyin
- all `item publish --no-download`: douyin
- all `item publish --allow-comment`: tiktok
- all `item publish --allow-duet`: tiktok
- all `item publish --allow-stitch`: tiktok
- all `item publish --allow-content-reuse`: tiktok
- all `item publish --allow-ai-remix`: tiktok
- all `item publish --cover43`: bilibili
- all `item publish --source`: bilibili
- all `item publish --dynamic`: bilibili
- all `item publish --allow-reprint`: bilibili
- all `item publish --shipping`: xianyu
- all `item publish --postage`: xianyu
- all `item publish --pickup`: xianyu
- all `item publish --original-price`: xianyu
- all `item publish --quote`: x
- all `item publish --thread`: x
- douyin `product get`: the upstream product API returns detail images, specs and a jump link only: title, image and price, with the other fields empty
- xhs `comment list`: --product is planned
- bilibili `comment list`: supports videos, articles, text / repost dynamics and image dynamics; image dynamics get the album comment-section id from a non-upstream dynamic detail API, with no fixture comparison
- kuaishou `comment list`: --product is planned
- weibo `comment list`: first-level comments only
- jd `comment list`: first page only; --limit N takes N comments in one request
- bilibili `comment replies`: not upstream: the nested-reply and dynamic-detail APIs, with no fixture comparison; covers visible replies under video, article and image-dynamic root comments; nested replies for text and repost dynamics are not verified and unsupported
- kuaishou `comment replies`: the upstream nested-comment API was checked against one captured parameter set only; results for other comments were not checked upstream
- youtube `comment replies`: <comment> is a root comment id; YouTube replies are one level deep
- bilibili `comment add`: dynamics support text and repost only: image dynamics (with images) hang their comment section on an album and upstream has no album-id API; videos and articles are unaffected
- youtube `comment add`: --reply-to takes a root comment or a reply; all replies hang under the root
- bilibili `comment delete`: dynamics support text and repost only: image dynamics (with images) hang their comment section on an album and upstream has no album-id API; videos and articles are unaffected
- youtube `comment delete`: own comments only
- all `comment list --product`: xhs, douyin, tiktok, kuaishou
- all `comment list --label`: douyin
- all `comment add --root`: bilibili
- xhs `feed list`: following is planned
- douyin `feed list`: hot and following are planned
- bilibili `feed list`: following is not upstream: upstream has no feed API, so the request follows the web dynamic page (x/polymer/web-dynamic/v1/feed/all) with no fixture comparison; lists videos from people you follow and needs a session
- kuaishou `feed list`: following is first screen only
- x `feed list`: following is planned
- youtube `feed list`: home recommendations and the subscription feed; the guest home is empty, so a session is needed
- maoyan `feed list`: returns every hot show in that city at once, unpaged; --category filters locally
- all `feed list --category`: xhs, maoyan, mteam
- all `feed list --city`: maoyan
- all `feed list --mode`: mteam
- all `feed list --free`: mteam
- all `feed list --order-by`: mteam
- all `feed list --asc`: mteam
- youtube `notice count`: unread total only, no breakdown
- xhs `msg list`: includes groups; a group conversation id is group:<group id>
- x `msg list`: inbox first page only (last 20 conversations); upstream has no paging
- mteam `msg list`: read-only and needs a password session; --box picks inbox / system / sent
- xhs `msg history`: direct chats take the peer id, groups take group:<group id>
- x `msg history`: messages are end-to-end encrypted, so only placeholder messages are emitted
- mteam `msg history`: <conversation> is an id from msg list; it is not marked read
- tiktok `msg send`: text only, and only to a user with an existing conversation (upstream cannot open one)
- xianyu `msg send`: without --item, --to opens the conversation on the upstream hard-coded default item; with --item, --to contacts that user about that item (a seller can reach a buyer)
- taobao `msg send`: --item contacts the product seller and --conversation replies into an existing conversation; --to is unsupported (there is no API to open a conversation by user)
- all `notice list --group`: douyin
- all `msg list --box`: mteam
- all `msg send --file`: douyin
- all `msg send --share`: douyin
- all `msg send --order`: jd
- xhs `folder list`: lists public collections only (upstream has that one API); upstream has no items-in-collection API
- bilibili `folder list`: lists only collections the user created, all in one response
- bilibili `folder items`: not upstream: upstream has no collection-content API, so the request follows the web collection page (x/v3/fav/resource/list) with no fixture comparison; videos only
- youtube `series list`: the channel's public playlists; me is all your playlists (needs a session)
- youtube `series items`: <series> is a playlist id (PL…) or a link carrying list=
- tiktok `poi search`: filters the recommended places by keyword
- maoyan `poi search`: searches show cities only (built-in city table); the id is the feed list --city value
- all `history list --area`: jd
- youtube `live get`: <room> is the live video id or link
- tiktok `live list`: only followed people who are live (upstream get_webcast_feed is the follow list in the live page sidebar); empty when nobody followed is live
- youtube `live listen`: polls the live chat: text messages, Super Chat, memberships
- douyin `live history`: only the last 15 messages fetched when entering the room
- tiktok `live history`: only the danmaku fetched when entering the room (im/fetch)
- xhs `live send`: --gift is planned
- douyin `live send`: --gift is planned
- tiktok `live send`: --gift is planned
- youtube `live send`: text only; --gift is unsupported
- douyin `live media`: the stream URL comes from the room profile (room/web/enter)
- tiktok `live media`: upstream does not resolve the stream URL: take liveRoom.streamData from /api-live/user/room, or stream_url from room/enter
- all `live list --category`: xhs
- all `live send --color`: bilibili
- all `live send --font-size`: bilibili
- all `live send --position`: bilibili
- all `live send --reply-user`: bilibili
- all `live like --count`: douyin
- all `live rank --ranking`: douyin
- all `live start --category`: bilibili

## Login methods

| 命令 | xhs | douyin | tiktok | bilibili | kuaishou | weibo | xianyu | taobao | jd | x | youtube | 12306 | jimeng | maoyan | mteam |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| apikey |  |  |  |  |  |  |  |  |  |  |  |  |  |  | ✓ |
| cookie | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |  |  | ✓ |  |
| oauth |  |  |  |  |  |  |  |  |  |  |  |  | ✓ |  |  |
| password |  |  |  | ✓ |  |  |  |  |  |  |  |  |  |  | ✓ |
| qrcode | ✓ | ✓ |  | ✓ | ✓ |  | ✓ |  | ✓ |  |  |  |  | ✓ |  |
| sms | ✓ | ✓ |  | ✓ | ✓ |  |  |  | ✓ |  |  |  |  |  |  |
