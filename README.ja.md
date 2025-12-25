# loadshow

[English README](README.md)

Webページの読み込みをMP4動画として記録するCLIツール・Goライブラリです。Webパフォーマンスの可視化に使用できます。

## 特徴

- Webページの読み込みプロセスをスクロール動画として記録
- AV1エンコード（libaom）による高品質・小ファイルサイズ
- デスクトップ/モバイルのプリセット設定
- ネットワークスロットリング（低速回線のシミュレーション）
- CPUスロットリング（低性能デバイスのシミュレーション）
- Juxtaposeコマンドで2つの動画を横並びで比較
- レイアウト、色、スタイルのカスタマイズ
- クロスプラットフォーム対応：Linux、macOS、Windows
- CLIツールとしてもGoライブラリとしても利用可能

## インストール

### バイナリのダウンロード

[GitHub Releases](https://github.com/user/loadshow/releases)から最新版をダウンロードしてください。

```bash
# Linux (amd64)
curl -LO https://github.com/user/loadshow/releases/latest/download/loadshow_vX.X.X_linux_amd64.tar.gz
tar -xzf loadshow_vX.X.X_linux_amd64.tar.gz
sudo mv loadshow /usr/local/bin/

# macOS (arm64)
curl -LO https://github.com/user/loadshow/releases/latest/download/loadshow_vX.X.X_darwin_arm64.tar.gz
tar -xzf loadshow_vX.X.X_darwin_arm64.tar.gz
sudo mv loadshow /usr/local/bin/
```

### ソースからビルド

Go 1.21以上とlibaomが必要です。

```bash
# 依存関係のインストール
make deps

# ビルド
make build
```

## 動作要件

- Chrome または Chromium ブラウザ（自動検出、または `CHROME_PATH` 環境変数で指定）

## CLIの使い方

### コマンド一覧

```
loadshow record <url> -o <output>     Webページの読み込みをMP4動画として記録
loadshow juxtapose <left> <right> -o <output>  2つの動画を横並びで比較
loadshow version                       バージョン情報を表示
```

### 基本的な記録

```bash
# モバイルプリセットでページを記録（デフォルト）
loadshow record https://example.com -o output.mp4

# デスクトッププリセットで記録
loadshow record https://example.com -o output.mp4 -p desktop
```

### 品質プリセット

```bash
# 低品質（高速、小ファイルサイズ）
loadshow record https://example.com -o output.mp4 -q low

# 高品質（低速、大ファイルサイズ）
loadshow record https://example.com -o output.mp4 -q high

# カスタムCRF値（0-63、低いほど高品質、プリセットを上書き）
loadshow record https://example.com -o output.mp4 --video-crf 20

# カスタムスクリーンキャスト品質（0-100、プリセットを上書き）
loadshow record https://example.com -o output.mp4 --screencast-quality 90
```

### 動画オプション

```bash
# カスタム動画サイズ
loadshow record https://example.com -o output.mp4 -W 640 -H 480
```

### ネットワークスロットリング

```bash
# 低速接続をシミュレート（1.5 Mbps）
loadshow record https://example.com -o output.mp4 --download-mbps 1.5

# 低速アップロードをシミュレート（0.5 Mbps）
loadshow record https://example.com -o output.mp4 --upload-mbps 0.5
```

### CPUスロットリング

```bash
# 4倍遅いCPUをシミュレート
loadshow record https://example.com -o output.mp4 --cpu-throttling 4.0
```

### レイアウトのカスタマイズ

```bash
# カラム数と間隔の指定
loadshow record https://example.com -o output.mp4 -c 3 --gap 10 --margin 20

# カスタム色
loadshow record https://example.com -o output.mp4 --background-color "#f0f0f0" --border-color "#cccccc"
```

### ブラウザオプション

```bash
# Chromeのパスを指定
loadshow record https://example.com -o output.mp4 --chrome-path /path/to/chrome

# ヘッドレスモードを無効化（ブラウザを表示）
loadshow record https://example.com -o output.mp4 --no-headless

# HTTPS証明書エラーを無視
loadshow record https://example.com -o output.mp4 --ignore-https-errors

# プロキシを使用
loadshow record https://example.com -o output.mp4 --proxy-server http://proxy:8080
```

### Juxtapose（横並び比較）

```bash
# 2つの動画を横並びで比較
loadshow juxtapose before.mp4 after.mp4 -o comparison.mp4
```

### デバッグモード

```bash
# デバッグ出力を有効化（中間フレームを保存）
loadshow record https://example.com -o output.mp4 -d --debug-dir ./debug
```

## 全オプション一覧

### record

```
使用法: loadshow record <url> -o <output> [flags]

引数:
  <url>    記録するページのURL

フラグ:
  出力先:
    -o, --output STRING        出力MP4ファイルパス（必須）

  プリセット:
    -p, --preset STRING        デバイスプリセット: desktop, mobile（デフォルト: mobile）
    -q, --quality STRING       品質プリセット: low, medium, high（デフォルト: medium）

  ブラウザ設定:
        --viewport-width INT   ブラウザビューポート幅（最小: 500）
        --chrome-path STRING   Chrome実行ファイルのパス
        --no-headless          ブラウザを非ヘッドレスモードで実行
        --no-incognito         シークレットモードを無効化
        --ignore-https-errors  HTTPS証明書エラーを無視
        --proxy-server STRING  HTTPプロキシサーバー（例: http://proxy:8080）

  性能エミュレーション:
        --download-mbps FLOAT  ダウンロード速度（Mbps、0 = 無制限）
        --upload-mbps FLOAT    アップロード速度（Mbps、0 = 無制限）
        --cpu-throttling FLOAT CPUスローダウン係数（1.0 = 制限なし）

  レイアウトとスタイル:
    -c, --columns INT          カラム数（最小: 1）
        --margin INT           キャンバス周りの余白（ピクセル）
        --gap INT              カラム間の間隔（ピクセル）
        --indent INT           2列目以降の追加上余白
        --outdent INT          1列目の追加下余白
        --background-color STR 背景色（16進数、例: #dcdcdc）
        --border-color STR     枠線色（16進数、例: #b4b4b4）
        --border-width INT     枠線幅（ピクセル）

  バナー:
        --credit STRING        バナーに表示するカスタムテキスト

  動画と品質:
    -W, --width INT            出力動画の幅
    -H, --height INT           出力動画の高さ
        --video-crf INT        動画CRF値（0-63、品質プリセットを上書き）
        --screencast-quality INT  スクリーンキャストJPEG品質（0-100、プリセットを上書き）
        --outro-ms INT         最終フレーム保持時間（ミリ秒）

  デバッグ:
    -d, --debug                デバッグ出力を有効化
        --debug-dir STRING     デバッグ出力ディレクトリ（デフォルト: ./debug）

  ログ:
    -l, --log-level STRING     ログレベル: debug, info, warn, error（デフォルト: info）
    -Q, --quiet                全てのログ出力を抑制
```

### juxtapose

```
使用法: loadshow juxtapose <left> <right> -o <output> [flags]

引数:
  <left>   左側の動画ファイルパス
  <right>  右側の動画ファイルパス

フラグ:
  出力先:
    -o, --output STRING    出力MP4ファイルパス（必須）

  プリセット:
    -q, --quality STRING   品質プリセット: low, medium, high（デフォルト: medium）

  レイアウトとスタイル:
        --gap INT          動画間の隙間（ピクセル、デフォルト: 10）

  動画と品質:
        --video-crf INT    動画CRF値（0-63、品質プリセットを上書き）
```

## GoライブラリとしてのAPI利用

loadshowはGoライブラリとしてプログラムから動画生成を行うことも可能です。

### インストール

```bash
go get github.com/user/loadshow
```

### ConfigBuilderを使った基本的な使い方

```go
package main

import (
    "context"
    "log"
    "runtime"

    "github.com/user/loadshow/pkg/adapters/av1encoder"
    "github.com/user/loadshow/pkg/adapters/chromebrowser"
    "github.com/user/loadshow/pkg/adapters/capturehtml"
    "github.com/user/loadshow/pkg/adapters/filesink"
    "github.com/user/loadshow/pkg/adapters/ggrenderer"
    "github.com/user/loadshow/pkg/adapters/logger"
    "github.com/user/loadshow/pkg/adapters/nullsink"
    "github.com/user/loadshow/pkg/adapters/osfilesystem"
    "github.com/user/loadshow/pkg/loadshow"
    "github.com/user/loadshow/pkg/orchestrator"
    "github.com/user/loadshow/pkg/ports"
    "github.com/user/loadshow/pkg/stages/banner"
    "github.com/user/loadshow/pkg/stages/composite"
    "github.com/user/loadshow/pkg/stages/encode"
    "github.com/user/loadshow/pkg/stages/layout"
    "github.com/user/loadshow/pkg/stages/record"
)

func main() {
    // モバイルプリセットで設定を作成（デフォルト）
    cfg := loadshow.NewConfigBuilder().
        WithWidth(512).
        WithHeight(640).
        WithColumns(3).
        WithVideoCRF(30).
        Build()

    // またはデスクトッププリセットを使用
    // cfg := loadshow.NewDesktopConfigBuilder().Build()

    // アダプタを作成
    fs := osfilesystem.New()
    renderer := ggrenderer.New()
    browser := chromebrowser.New()
    htmlCapturer := capturehtml.New()
    encoder := av1encoder.New()
    sink := nullsink.New()
    log := logger.NewConsole(ports.LogLevelInfo)

    // パイプラインステージを作成
    layoutStage := layout.NewStage()
    recordStage := record.New(browser, sink, log, ports.BrowserOptions{
        Headless:  true,
        Incognito: true,
    })
    bannerStage := banner.NewStage(htmlCapturer, sink, log)
    compositeStage := composite.NewStage(renderer, sink, log, runtime.NumCPU())
    encodeStage := encode.NewStage(encoder, log)

    // オーケストレータを作成して実行
    orch := orchestrator.New(
        layoutStage,
        recordStage,
        bannerStage,
        compositeStage,
        encodeStage,
        fs,
        sink,
        log,
    )

    orchConfig := cfg.ToOrchestratorConfig("https://example.com", "output.mp4")
    if err := orch.Run(context.Background(), orchConfig); err != nil {
        log.Fatal(err)
    }
}
```

### ConfigBuilderのメソッド一覧

```go
// 動画サイズ
builder.WithWidth(512)           // 出力動画の幅
builder.WithHeight(640)          // 出力動画の高さ

// レイアウトオプション
builder.WithViewportWidth(375)   // ブラウザビューポート幅（最小: 500）
builder.WithColumns(3)           // カラム数（最小: 1）
builder.WithMargin(20)           // キャンバス周りの余白
builder.WithGap(20)              // カラム間の間隔
builder.WithIndent(20)           // 2列目以降の上余白
builder.WithOutdent(20)          // 1列目の下余白

// スタイルオプション
builder.WithBackgroundColor(color.RGBA{220, 220, 220, 255})
builder.WithBorderColor(color.RGBA{180, 180, 180, 255})
builder.WithBorderWidth(1)

// エンコードオプション
builder.WithVideoCRF(30)         // 動画CRF 0-63（低いほど高品質）
builder.WithScreencastQuality(80) // スクリーンキャストJPEG品質 0-100
builder.WithOutroMs(2000)        // 最終フレーム保持時間

// ネットワークスロットリング
builder.WithDownloadSpeed(loadshow.Mbps(10))  // 10 Mbps
builder.WithUploadSpeed(loadshow.Mbps(5))     // 5 Mbps
builder.WithNetworkSpeed(loadshow.Mbps(10))   // 上下両方向

// CPUスロットリング
builder.WithCPUThrottling(4.0)   // 4倍遅い

// ブラウザオプション
builder.WithIgnoreHTTPSErrors(true)
builder.WithProxyServer("http://proxy:8080")

// バナー
builder.WithCredit("会社名")
```

### Juxtapose API

```go
package main

import (
    "context"
    "log"

    "github.com/user/loadshow/pkg/adapters/av1decoder"
    "github.com/user/loadshow/pkg/adapters/av1encoder"
    "github.com/user/loadshow/pkg/adapters/logger"
    "github.com/user/loadshow/pkg/adapters/osfilesystem"
    "github.com/user/loadshow/pkg/juxtapose"
)

func main() {
    // シンプルな関数呼び出し
    err := juxtapose.Combine(
        "before.mp4",
        "after.mp4",
        "comparison.mp4",
        juxtapose.DefaultOptions(),
    )
    if err != nil {
        log.Fatal(err)
    }

    // またはStage APIを使用してより詳細に制御
    decoder := av1decoder.NewMP4Reader()
    defer decoder.Close()

    encoder := av1encoder.New()
    fs := osfilesystem.New()
    log := logger.NewConsole(ports.LogLevelInfo)

    opts := juxtapose.Options{
        Gap:     10,      // 動画間の隙間
        FPS:     30.0,    // 出力フレームレート
        Quality: 30,      // CRF品質
        Bitrate: 0,       // 自動ビットレート
    }

    stage := juxtapose.New(decoder, encoder, fs, log, opts)
    result, err := stage.Execute(context.Background(), juxtapose.Input{
        LeftPath:   "before.mp4",
        RightPath:  "after.mp4",
        OutputPath: "comparison.mp4",
    })
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("作成フレーム数: %d, 再生時間: %dms", result.FrameCount, result.DurationMs)
}
```

## 開発

```bash
# 依存関係のインストール（動的リンク、開発用）
make deps

# ビルド
make build

# テスト実行
make test

# E2Eを含む全テスト実行
make test-all

# 利用可能なターゲット一覧
make help
```

### リリースビルド

```bash
# 依存関係のインストール（静的リンク）
make deps-static

# 静的バイナリをバージョン付きでビルド
make build-static VERSION=v1.0.0

# リリースアーカイブを作成
make package VERSION=v1.0.0
```

## アーキテクチャ

loadshowは依存性注入を用いたパイプラインアーキテクチャを採用しています：

```
┌─────────────────────────────────────────────────────────────┐
│                      Orchestrator                           │
├─────────────────────────────────────────────────────────────┤
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌───────────────┐│
│  │  Layout  │→ │  Record  │→ │  Banner  │→ │   Composite   ││
│  │  Stage   │  │  Stage   │  │  Stage   │  │     Stage     ││
│  └──────────┘  └──────────┘  └──────────┘  └───────────────┘│
│                                                      ↓      │
│                                             ┌───────────────┐│
│                                             │    Encode     ││
│                                             │     Stage     ││
│                                             └───────────────┘│
└─────────────────────────────────────────────────────────────┘
```

1. **Layout Stage** - 設定に基づいて動画レイアウトを計算
2. **Record Stage** - Chrome DevTools Protocolを使用してページ読み込み中のスクリーンショットを取得
3. **Banner Stage** - タイミング情報を含む情報バナーを生成
4. **Composite Stage** - スクリーンショットを動画フレームにレンダリング
5. **Encode Stage** - フレームをAV1/MP4にエンコード

### パッケージ構造

```
pkg/
├── loadshow/        # ConfigBuilderを含む高レベルAPI
├── orchestrator/    # パイプライン調整
├── pipeline/        # ステージインターフェースと型
├── stages/          # パイプラインステージ実装
│   ├── layout/      # レイアウト計算
│   ├── record/      # ページ記録
│   ├── banner/      # バナー生成
│   ├── composite/   # フレーム合成
│   └── encode/      # 動画エンコード
├── ports/           # インターフェース定義（ポート）
├── adapters/        # インターフェース実装（アダプタ）
│   ├── av1encoder/  # AV1動画エンコード
│   ├── av1decoder/  # AV1動画デコード
│   ├── chromebrowser/
│   ├── ggrenderer/
│   └── ...
├── juxtapose/       # 横並び動画比較
└── mocks/           # テスト用モック
```

## ライセンス

MIT License
