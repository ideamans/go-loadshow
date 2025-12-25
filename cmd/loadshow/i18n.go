// Package main provides localization for the loadshow CLI.
package main

import (
	"github.com/ideamans/go-l10n"
)

func init() {
	// Register Japanese translations for CLI messages.
	l10n.Register("ja", l10n.LexiconMap{
		// Flag categories
		"Output":               "出力先",
		"Preset":               "プリセット",
		"Browser":              "ブラウザ設定",
		"Performance Emulation": "性能エミュレーション",
		"Layout and Style":     "レイアウトとスタイル",
		"Banner":               "バナー",
		"Video and Quality":    "動画と品質",
		"Debug":                "デバッグ",
		"Logging":              "ログ",

		// Root command
		"Create page load videos for web performance visualization":            "Webページの読み込みパフォーマンスを可視化する動画を作成",
		"loadshow creates videos that visualize web page loading performance.": "loadshowはWebページの読み込みパフォーマンスを可視化する動画を作成します。",

		// Record command
		"Record a web page loading as MP4 video":                                "Webページの読み込みをMP4動画として記録",
		"Record the loading process of a web page and save it as an MP4 video.": "Webページの読み込み過程を記録し、MP4動画として保存します。",

		// Juxtapose command
		"Create a side-by-side comparison video":                        "2つの動画を並べた比較動画を作成",
		"Create a side-by-side comparison video from two input videos.": "2つの入力動画から並列比較動画を作成します。",

		// Version command
		"Show version information":         "バージョン情報を表示",
		"Display the version of loadshow.": "loadshowのバージョンを表示します。",
		"loadshow (Go) version %s":         "loadshow (Go版) バージョン %s",

		// Required flags
		"Output MP4 file path (required)": "出力MP4ファイルパス（必須）",

		// Preset flags
		"Device preset (desktop, mobile)":  "デバイスプリセット（desktop, mobile）",
		"Quality preset (low, medium, high)": "品質プリセット（low, medium, high）",

		// Video output flags
		"Output video width (default: 512)":                              "出力動画の幅（デフォルト: 512）",
		"Output video height (default: 640)":                             "出力動画の高さ（デフォルト: 640）",
		"Video CRF value (0-63, lower is better, overrides quality preset)": "動画のCRF値（0-63、低いほど高品質、品質プリセットを上書き）",
		"Duration to hold final frame in milliseconds":                   "最終フレームの保持時間（ミリ秒）",

		// Recording flags
		"Screencast JPEG quality (0-100, overrides quality preset)": "スクリーンキャストのJPEG品質（0-100、品質プリセットを上書き）",
		"Browser viewport width (min: 500)":                             "ブラウザのビューポート幅（最小: 500）",

		// Layout flags
		"Number of columns (min: 1)":               "カラム数（最小: 1）",
		"Margin around the canvas in pixels":      "キャンバス周囲の余白（ピクセル）",
		"Gap between columns in pixels":           "カラム間の隙間（ピクセル）",
		"Additional top margin for columns 2+":    "2列目以降の追加上部余白",
		"Additional bottom margin for column 1":   "1列目の追加下部余白",

		// Style flags
		"Background color (hex, e.g., #dcdcdc)": "背景色（16進数、例: #dcdcdc）",
		"Border color (hex, e.g., #b4b4b4)":     "枠線の色（16進数、例: #b4b4b4）",
		"Border width in pixels":                "枠線の幅（ピクセル）",

		// Network throttling flags
		"Download speed in Mbps (0 = unlimited)": "ダウンロード速度（Mbps、0 = 無制限）",
		"Upload speed in Mbps (0 = unlimited)":   "アップロード速度（Mbps、0 = 無制限）",

		// CPU throttling flags
		"CPU slowdown factor (1.0 = no throttling, 4.0 = 4x slower)": "CPUスローダウン係数（1.0 = 制限なし、4.0 = 4倍遅く）",

		// Banner flags
		"Custom text shown in banner (default: loadshow)": "バナーに表示するカスタムテキスト（デフォルト: loadshow）",

		// Browser flags
		"Run browser in non-headless mode":          "ブラウザを非ヘッドレスモードで実行",
		"Path to Chrome executable":                 "Chrome実行ファイルのパス",
		"Ignore HTTPS certificate errors":           "HTTPS証明書エラーを無視",
		"HTTP proxy server (e.g., http://proxy:8080)": "HTTPプロキシサーバー（例: http://proxy:8080）",
		"Disable incognito mode":                    "シークレットモードを無効化",

		// Debug flags
		"Enable debug output":        "デバッグ出力を有効化",
		"Directory for debug output": "デバッグ出力のディレクトリ",

		// Logging flags
		"Log level (debug, info, warn, error)": "ログレベル（debug, info, warn, error）",
		"Suppress all log output":              "全てのログ出力を抑制",

		// Runtime messages
		"Recording %s (%s preset)...":   "%s を記録中 (%s プリセット)...",
		"Output saved to %s":            "出力を %s に保存しました",
		"Interrupted, shutting down...": "中断されました。シャットダウン中...",

		// Juxtapose messages
		"Juxtapose command not yet implemented.":       "Juxtaposeコマンドはまだ実装されていません。",
		"Would create comparison from %s and %s to %s": "%s と %s から %s への比較動画を作成します",

		// Orchestrator messages
		"Encoding video with CRF %d": "CRF %d で動画をエンコード中",

		// Error messages
		"URL argument is required":       "URL引数が必要です",
		"Two video arguments are required": "2つの動画引数が必要です",
	})
}
