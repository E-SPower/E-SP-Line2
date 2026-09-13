// Minimal EventToken.h compatibility header for cross-compiling
// github.com/webview/webview_go on Linux -> Windows (mingw-w64).
//
// Background:
//   webview_go vendors a trimmed copy of the WebView2 SDK headers, but
//   libs/mswebview2/include/WebView2.h does `#include "EventToken.h"`
//   while EventToken.h itself is NOT shipped with the module. MSVC builds
//   usually pick it up from the Windows SDK, so the omission goes unnoticed
//   on native Windows builds -- but a mingw-w64 cross build fails with:
//       fatal error: EventToken.h: No such file or directory
//
//   EventToken.h is a tiny MIDL-generated header whose only job is to define
//   the EventRegistrationToken struct used throughout WebView2.h. We provide
//   an ABI-compatible definition here and inject this directory via
//   CGO_CXXFLAGS so the third-party module source stays untouched.
//
//   The layout below matches the Windows SDK definition:
//       typedef struct EventRegistrationToken { __int64 value; } EventRegistrationToken;

#ifndef E_SP_LINE2_EVENTTOKEN_COMPAT_H_
#define E_SP_LINE2_EVENTTOKEN_COMPAT_H_

#if defined(_MSC_VER) || defined(__MINGW32__) || defined(__MINGW64__)
typedef struct EventRegistrationToken
{
    __int64 value;
} EventRegistrationToken;
#else
typedef struct EventRegistrationToken
{
    long long value;
} EventRegistrationToken;
#endif

#endif // E_SP_LINE2_EVENTTOKEN_COMPAT_H_
