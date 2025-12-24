# loadshow

[English README](README.md)

Webページの読み込みをMP4動画として記録するCLIツールです。Webパフォーマンスの可視化に使用できます。

## 特徴

- Webページの読み込みプロセスをスクロール動画として記録
- AV1エンコード（libaom）による高品質・小ファイルサイズ
- デスクトップ/モバイルのプリセット設定
- ネットワークスロットリング（低速回線のシミュレーション）
- CPUスロットリング（低性能デバイスのシミュレーション）
- レイアウト、色、スタイルのカスタマイズ
- クロスプラットフォーム対応：Linux、macOS、Windows

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

## 使い方

### 基本的な記録

```bash
# デスクトッププリセットでページを記録
loadshow record https://example.com -o output.mp4

# モバイルプリセットで記録
loadshow record https://example.com -o output.mp4 -p mobile
```

### 動画オプション

```bash
# カスタム動画サイズ
loadshow record https://example.com -o output.mp4 -W 640 -H 480

# 高画質（CRFが低いほど高品質、ファイルサイズ増）
loadshow record https://example.com -o output.mp4 -q 20
```

### ネットワークスロットリング

```bash
# 低速3G接続をシミュレート（50KB/s）
loadshow record https://example.com -o output.mp4 --download-speed 51200
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

### デバッグモード

```bash
# デバッグ出力を有効化（中間フレームを保存）
loadshow record https://example.com -o output.mp4 -d --debug-dir ./debug
```

## 全オプション一覧

```
使用法: loadshow record <url> -o <output> [flags]

引数:
  <url>    記録するページのURL

フラグ:
  -o, --output=STRING          出力MP4ファイルパス（必須）
  -p, --preset="desktop"       プリセット: desktop または mobile
  -W, --width=INT              出力動画の幅
  -H, --height=INT             出力動画の高さ
      --viewport-width=INT     ブラウザビューポート幅
  -c, --columns=INT            カラム数
      --margin=INT             キャンバス周りの余白
      --gap=INT                カラム間の間隔
      --indent=INT             2列目以降の上余白
      --outdent=INT            1列目の下余白
      --background-color=STR   背景色（16進数）
      --border-color=STR       枠線色（16進数）
      --border-width=INT       枠線幅（ピクセル）
  -q, --quality=INT            動画品質（CRF 0-63）
      --outro-ms=INT           最終フレーム保持時間（ミリ秒）
      --credit=STRING          バナーに表示するテキスト
      --download-speed=INT     ダウンロード速度制限（bytes/sec）
      --upload-speed=INT       アップロード速度制限（bytes/sec）
      --cpu-throttling=FLOAT   CPU速度低下係数
  -d, --debug                  デバッグ出力を有効化
      --debug-dir=STRING       デバッグ出力ディレクトリ
      --no-headless            ブラウザを表示
      --chrome-path=STRING     Chromeのパス
      --ignore-https-errors    証明書エラーを無視
      --proxy-server=STRING    HTTPプロキシサーバー
      --no-incognito           シークレットモードを無効化
  -l, --log-level="info"       ログレベル: debug,info,warn,error
  -Q, --quiet                  ログ出力を抑制
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

loadshowはパイプラインアーキテクチャを採用しています：

1. **Layout Stage** - 設定に基づいて動画レイアウトを計算
2. **Record Stage** - Chrome DevTools Protocolを使用してページ読み込み中のスクリーンショットを取得
3. **Banner Stage** - タイミング情報を含む情報バナーを生成
4. **Composite Stage** - スクリーンショットを動画フレームにレンダリング
5. **Encode Stage** - フレームをAV1/MP4にエンコード

## ライセンス

MIT License
