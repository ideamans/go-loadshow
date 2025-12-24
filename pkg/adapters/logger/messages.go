package logger

import "github.com/ideamans/go-l10n"

func init() {
	l10n.Register("ja", l10n.LexiconMap{
		// Orchestration level messages (info)
		"Recording %s (%s preset)...":      "%s を録画中 (%s プリセット)...",
		"Output saved to %s":               "出力を %s に保存しました",
		"Pipeline completed successfully":  "パイプラインが正常に完了しました",
		"Starting pipeline":                "パイプラインを開始します",
		"Interrupted, shutting down...":    "中断されました。シャットダウン中...",

		// Layout stage
		"Calculating layout":               "レイアウトを計算中",
		"Layout calculated: %dx%d canvas, %d columns": "レイアウト計算完了: %dx%d キャンバス, %d カラム",

		// Record stage (browser component)
		"Launching browser":                "ブラウザを起動中",
		"Launching browser in headless mode":    "ヘッドレスモードでブラウザを起動中",
		"Launching browser in visible mode":     "表示モードでブラウザを起動中",
		"Navigating to %s":                 "%s へ移動中",
		"Setting network conditions: %d ms latency, %d bps down, %d bps up": "ネットワーク条件を設定: レイテンシ %d ms, ダウン %d bps, アップ %d bps",
		"Setting CPU throttling: %.1fx slowdown": "CPUスロットリングを設定: %.1f倍 減速",
		"Starting screencast":              "スクリーンキャストを開始",
		"Captured %d frames":               "%d フレームをキャプチャしました",
		"Recording completed in %d ms":     "録画が %d ms で完了しました",
		"Browser closed":                   "ブラウザを閉じました",

		// Banner stage
		"Generating banner":                "バナーを生成中",
		"Banner generated: %dx%d":          "バナー生成完了: %dx%d",

		// Composite stage
		"Compositing %d frames with %d workers": "%d フレームを %d ワーカーで合成中",
		"Compositing frame %d/%d":          "フレーム合成中 %d/%d",
		"Composition completed":            "合成が完了しました",

		// Encode stage
		"Encoding video with quality %d":   "品質 %d で動画をエンコード中",
		"Encoding %d frames at %.1f fps":   "%d フレームを %.1f fps でエンコード中",
		"Video encoded: %d bytes":          "動画エンコード完了: %d バイト",
		"Encoding completed":               "エンコードが完了しました",

		// Warnings
		"Frame capture timeout, using collected frames": "フレームキャプチャがタイムアウトしました。収集したフレームを使用します",
		"Some frames may be missing":       "一部のフレームが欠落している可能性があります",

		// Errors
		"Failed to launch browser: %s":     "ブラウザの起動に失敗しました: %s",
		"Failed to navigate: %s":           "ページ移動に失敗しました: %s",
		"Failed to encode video: %s":       "動画のエンコードに失敗しました: %s",
		"Failed to write output: %s":       "出力の書き込みに失敗しました: %s",
	})
}
