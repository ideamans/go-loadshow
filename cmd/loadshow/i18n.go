// Package main provides localization for the loadshow CLI.
package main

import (
	"github.com/ideamans/go-l10n"
)

func init() {
	// Register Japanese translations for CLI-specific messages.
	// Note: Logger/pipeline messages are in pkg/adapters/logger/messages.go
	l10n.Register("ja", l10n.LexiconMap{
		// Version command
		"loadshow (Go) version %s": "loadshow (Go版) バージョン %s",

		// Record command
		"Recording %s (%s preset)...":   "%s を記録中 (%s プリセット)...",
		"Output saved to %s":            "出力を %s に保存しました",
		"Interrupted, shutting down...": "中断されました。シャットダウン中...",

		// Juxtapose command
		"Juxtapose command not yet implemented.":       "Juxtaposeコマンドはまだ実装されていません。",
		"Would create comparison from %s and %s to %s": "%s と %s から %s への比較動画を作成します",

		// CLI-level errors
		"chrome not found: please install Chrome/Chromium, set CHROME_PATH environment variable, or use --chrome-path option": "Chromeが見つかりません: Chrome/Chromiumをインストールするか、CHROME_PATH環境変数を設定するか、--chrome-pathオプションを使用してください",
	})
}
