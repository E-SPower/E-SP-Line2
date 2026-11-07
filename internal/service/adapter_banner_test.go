package service

import (
	"strings"
	"testing"
)

// TestIsBannerLineRecognizesArt guards the level filter: the banner carries no
// log level, and before this check the default "info" view dropped it.
func TestIsBannerLineRecognizesArt(t *testing.T) {
	bannerLines := strings.Split(adapterBanner, "\n")
	art := 0
	for _, line := range bannerLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "Power By") || strings.HasPrefix(trimmed, "---") {
			continue
		}
		art++
		if !isBannerLine(line) {
			t.Fatalf("art line not recognized as banner: %q", line)
		}
		if !logLineMatches(line, "info") {
			t.Fatalf("banner art must survive the info filter: %q", line)
		}
	}
	if art == 0 {
		t.Fatal("adapterBanner has no art lines")
	}
	if !isBannerLine("  Power By LangBot-community-team") {
		t.Fatal("footer not recognized as banner")
	}
	if isBannerLine("2026-01-01 00:00:00 | INFO     | [start] x") {
		t.Fatal("a normal record must not be classified as banner")
	}
}

// TestStandaloneLogLinesSurviveInfoFilter pins that the standalone adapter's own
// test log lines (时间 | 级别 | [tag] 消息) are classified by level, so the
// default info filter keeps info/warning and the error filter finds errors.
func TestStandaloneLogLinesSurviveInfoFilter(t *testing.T) {
	info := "2026-11-07 13:00:00 | INFO     | [start] 独立式接入器启动"
	// "instance" contains "stance" — logLineLevel must match the level column,
	// not any substring.
	if !logLineMatches(info, "info") {
		t.Fatalf("info record filtered out: %q", info)
	}
	if logLineMatches(info, "error") {
		t.Fatalf("info record leaked into the error view: %q", info)
	}

	warn := "2026-11-07 13:00:01 | WARNING  | [catbus] 代理握手慢"
	if !logLineMatches(warn, "warning") {
		t.Fatalf("warning record filtered out: %q", warn)
	}

	errLine := "2026-11-07 13:00:02 | ERROR    | [ack] 发送失败 code=AUTH_REQUIRED"
	if !logLineMatches(errLine, "error") {
		t.Fatalf("error record filtered out: %q", errLine)
	}
	if logLineMatches(errLine, "info") {
		t.Fatalf("error record leaked into the info view: %q", errLine)
	}

	// The level column itself must be what matched.
	if got := logLineLevel(warn); got != "warning" {
		t.Fatalf("logLineLevel = %q, want warning", got)
	}
}

// TestAdapterBannerCoversStandalone keeps the two banners in lockstep: the
// runner echoes adapterBanner into every instance log, and the standalone
// adapter prints its own copy. They must stay the same art.
func TestAdapterBannerCoversStandalone(t *testing.T) {
	if !strings.Contains(adapterBanner, "███████╗") {
		t.Fatal("adapterBanner lost the ESPL art")
	}
	if !strings.Contains(adapterBanner, "Power By LangBot-community-team") {
		t.Fatal("adapterBanner lost the footer")
	}
	// The banner echoed by the runner must not be filtered at info level.
	for _, line := range strings.Split(adapterBanner, "\n") {
		if isBannerLine(line) && !logLineMatches(line, "info") {
			t.Fatalf("banner line filtered at info: %q", line)
		}
	}
}
