package logger

import "github.com/ideamans/go-l10n"

func init() {
	l10n.Register("ja", l10n.LexiconMap{
		// Pipeline orchestration messages
		"Starting pipeline":               "パイプラインを開始",
		"Pipeline completed successfully": "パイプラインが正常に完了しました",

		// Layout stage
		"Calculating layout":                              "レイアウトを計算中",
		"Layout calculated: %dx%d canvas, %d columns":     "レイアウト計算完了: %dx%d キャンバス, %d カラム",

		// Record stage (browser component)
		"Launching browser":                               "ブラウザを起動中",
		"Launching browser in headless mode":              "ヘッドレスモードでブラウザを起動中",
		"Launching browser in visible mode":               "表示モードでブラウザを起動中",
		"Navigating to %s":                                "%s へ移動中",
		"Setting network conditions: %d ms latency, %d bps down, %d bps up": "ネットワーク条件を設定: %d ms 遅延, %d bps ダウン, %d bps アップ",
		"Setting CPU throttling: %.1fx slowdown":          "CPUスロットリングを設定: %.1fx 低速化",
		"Starting screencast":                             "スクリーンキャストを開始",
		"Captured %d frames":                              "%d フレームをキャプチャしました",
		"Recording completed in %d ms":                    "記録が %d ms で完了しました",
		"Browser closed":                                  "ブラウザを閉じました",

		// Banner stage
		"Generating banner":    "バナーを生成中",
		"Banner generated: %dx%d": "バナー生成完了: %dx%d",

		// Composite stage
		"Compositing %d frames":                 "%d フレームを合成中",
		"Compositing %d frames with %d workers": "%d フレームを %d ワーカーで合成中",
		"Compositing frame %d/%d":               "フレーム %d/%d を合成中",
		"Composition completed":                 "合成完了",

		// Encode stage
		"Encoding video with quality %d": "品質 %d で動画をエンコード中",
		"Encoding %d frames at %.1f fps": "%d フレームを %.1f fps でエンコード中",
		"Video encoded: %d bytes":        "動画エンコード完了: %d バイト",
		"Encoding completed":             "エンコードが完了しました",

		// Warnings
		"Frame capture timeout, using collected frames": "フレームキャプチャがタイムアウトしました。収集したフレームを使用します",
		"Some frames may be missing":                    "一部のフレームが欠落している可能性があります",

		// Errors (pipeline level)
		"Failed to calculate layout: %s":  "レイアウト計算に失敗: %s",
		"Failed to record page: %s":       "ページ記録に失敗: %s",
		"Failed to generate banner: %s":   "バナー生成に失敗: %s",
		"Failed to composite frames: %s":  "フレーム合成に失敗: %s",
		"Failed to encode video: %s":      "動画エンコードに失敗: %s",
		"Failed to write output: %s":      "出力の書き込みに失敗: %s",
		"Failed to launch browser: %s":    "ブラウザの起動に失敗: %s",
		"Failed to navigate: %s":          "ページ移動に失敗: %s",
	})
}
