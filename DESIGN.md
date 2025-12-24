# Loadshow Go 実装設計書

## 概要

Loadshow を Go 言語のシングルバイナリとして再実装します。外部依存を抽象化し、各処理段階をテスタブルなパイプラインとして構成します。

## 主要機能

1. **Chrome 操作による Web ページの録画**
   - CDP（Chrome DevTools Protocol）経由でのブラウザ操作
   - ページ読み込みプロセスのスクリーンキャスト
   - ネットワーク条件や CPU スロットリングのエミュレーション

2. **縦長ページのレイアウト分割**
   - 長い Web ページを複数カラムに分割表示
   - スクロール位置の計算と管理

3. **フレーム合成**
   - バナー、プログレスバー、スクリーンショットの合成
   - 並列処理による高速化

4. **WebM 動画生成**
   - libvpx を使った VP8/VP9 エンコード
   - ffmpeg 不要のシングルバイナリ実現

## アーキテクチャ概要

### パイプラインアーキテクチャ

```text
[Config] → [LayoutStage] → [LayoutResult]
                                ↓
[URL] ──→ [RecordStage] → [RecordResult]
                                ↓
          [BannerStage] → [BannerResult]
                                ↓
        [CompositeStage] → [CompositeResult]
                                ↓
          [EncodeStage] → [EncodeResult] → [WebM File]
```

### レイヤー構成

```text
┌─────────────────────────────────────────────────────┐
│                  CLI (Kong)                         │
└─────────────────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────┐
│                 Orchestrator                        │
│   (各ステージの実行順序とデータフローを制御)           │
└─────────────────────────────────────────────────────┘
                        │
        ┌───────────────┼───────────────┐
        ▼               ▼               ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│    Stages    │ │    Stages    │ │    Stages    │
│   (Layout,   │ │   (Record,   │ │  (Composite, │
│    Banner)   │ │   Encode)    │ │   etc.)      │
└──────────────┘ └──────────────┘ └──────────────┘
        │               │               │
        ▼               ▼               ▼
┌─────────────────────────────────────────────────────┐
│                     Ports                           │
│  (Browser, Renderer, VideoEncoder, FileSystem)      │
└─────────────────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────┐
│                    Adapters                         │
│  (chromedp, gg, libvpx, os)                        │
└─────────────────────────────────────────────────────┘
```

## Ports（外部依存インターフェース）

### Browser - ブラウザ操作

```go
// pkg/ports/browser.go
type Browser interface {
    Launch(ctx context.Context, opts BrowserOptions) error
    Navigate(url string) error
    SetViewport(width, height int) error
    SetNetworkConditions(conditions NetworkConditions) error
    SetCPUThrottling(rate float64) error
    StartScreencast(quality int) (<-chan ScreenFrame, error)
    StopScreencast() error
    GetPageInfo() (*PageInfo, error)
    Close() error
}

type BrowserOptions struct {
    Headless    bool
    ChromePath  string
    UserAgent   string
    Headers     map[string]string
}

type NetworkConditions struct {
    Latency        int     // ms
    DownloadSpeed  int     // bytes/sec
    UploadSpeed    int     // bytes/sec
    Offline        bool
}

type ScreenFrame struct {
    TimestampMs int
    Data        []byte // JPEG
    Metadata    ScreenFrameMetadata
}

type PageInfo struct {
    Title        string
    URL          string
    ScrollHeight int
    ScrollWidth  int
}
```

### Renderer - 画像描画

```go
// pkg/ports/renderer.go
type Renderer interface {
    CreateCanvas(width, height int, bg color.Color) Canvas
    DecodeImage(data []byte, format ImageFormat) (image.Image, error)
    EncodeImage(img image.Image, format ImageFormat, quality int) ([]byte, error)
    ResizeImage(img image.Image, width, height int) image.Image
}

type Canvas interface {
    DrawImage(img image.Image, x, y int)
    DrawImageScaled(img image.Image, x, y, width, height int)
    DrawRect(x, y, w, h int, c color.Color)
    DrawRoundedRect(x, y, w, h, radius int, c color.Color)
    DrawText(text string, x, y int, style TextStyle)
    DrawLine(x1, y1, x2, y2 int, c color.Color, width float64)
    ToImage() image.Image
}

type TextStyle struct {
    FontSize  float64
    FontPath  string
    Color     color.Color
    Align     TextAlign
}

type ImageFormat int

const (
    FormatJPEG ImageFormat = iota
    FormatPNG
)
```

### VideoEncoder - 動画エンコーディング

```go
// pkg/ports/encoder.go
type VideoEncoder interface {
    Begin(width, height int, fps float64, opts EncoderOptions) error
    EncodeFrame(img image.Image, timestampMs int) error
    End() ([]byte, error) // WebMデータを返す
}

type EncoderOptions struct {
    Codec   VPXCodec
    Bitrate int // kbps
    Quality int // CRF: 0-63 (低いほど高品質)
}

type VPXCodec int

const (
    CodecVP8 VPXCodec = iota
    CodecVP9
)
```

### FileSystem - ファイル入出力

```go
// pkg/ports/filesystem.go
type FileSystem interface {
    ReadFile(path string) ([]byte, error)
    WriteFile(path string, data []byte) error
    MkdirAll(path string) error
    Exists(path string) (bool, error)
    Remove(path string) error
}
```

### DebugSink - デバッグ用中間出力

```go
// pkg/ports/sink.go
type DebugSink interface {
    SaveLayout(result LayoutResult) error
    SaveRawFrame(index int, frame RawFrame) error
    SaveBanner(img image.Image) error
    SaveComposedFrame(index int, img image.Image) error
    Enabled() bool
}
```

## Pipeline Types（パイプライン共通型）

### ステージインターフェース

```go
// pkg/pipeline/stage.go
type Stage[In, Out any] interface {
    Execute(ctx context.Context, input In) (Out, error)
}
```

### 各ステージの入出力型

```go
// pkg/pipeline/types.go

// === Layout Stage ===
type LayoutInput struct {
    CanvasWidth    int
    CanvasHeight   int
    Columns        int
    Gap            int
    Padding        int
    BorderWidth    int
    BannerHeight   int
    ProgressHeight int
}

type LayoutResult struct {
    ScrollDimensions Dimension     // 全体のスクロール領域
    ViewportSize     Dimension     // 各ウィンドウのビューポートサイズ
    Columns          []Rectangle   // カラムの位置とサイズ
    Windows          []Window      // ビューポートウィンドウ
    BannerArea       Rectangle     // バナー描画領域
    ProgressArea     Rectangle     // プログレスバー描画領域
    ContentArea      Rectangle     // コンテンツ描画領域
}

type Dimension struct {
    Width  int
    Height int
}

type Rectangle struct {
    X, Y, Width, Height int
}

type Window struct {
    ColumnIndex  int
    ScrollY      int
    ViewportRect Rectangle
}

// === Record Stage ===
type RecordInput struct {
    URL               string
    ViewportWidth     int
    ViewportHeight    int
    TimeoutMs         int
    NetworkConditions NetworkConditions
    CPUThrottling     float64
    Headers           map[string]string
}

type RecordResult struct {
    Frames   []RawFrame
    PageInfo PageInfo
    Timing   TimingInfo
}

type RawFrame struct {
    TimestampMs     int
    ImageData       []byte // JPEG
    LoadedResources int
    TotalResources  int
}

type TimingInfo struct {
    NavigationStartMs int
    DOMContentLoadedMs int
    LoadCompleteMs    int
    TotalDurationMs   int
}

// === Banner Stage ===
type BannerInput struct {
    Width      int
    Height     int
    URL        string
    Title      string
    LoadTimeMs int
    TotalBytes int64
    Theme      BannerTheme
}

type BannerTheme struct {
    BackgroundColor color.Color
    TextColor       color.Color
    AccentColor     color.Color
}

type BannerResult struct {
    Image image.Image
}

// === Composite Stage ===
type CompositeInput struct {
    RawFrames    []RawFrame
    Layout       LayoutResult
    Banner       *BannerResult // optional
    Theme        CompositeTheme
    ShowProgress bool
}

type CompositeTheme struct {
    BackgroundColor  color.Color
    BorderColor      color.Color
    ProgressBarColor color.Color
}

type CompositeResult struct {
    Frames []ComposedFrame
}

type ComposedFrame struct {
    TimestampMs int
    Image       image.Image
}

// === Encode Stage ===
type EncodeInput struct {
    Frames  []ComposedFrame
    OutroMs int
    Codec   VPXCodec
    Quality int
    Bitrate int
    FPS     float64
}

type EncodeResult struct {
    WebMData   []byte
    DurationMs int
    FileSize   int64
}
```

## Stages（各ステージ実装）

### 1. Layout Stage

**責務**: レイアウト計算（純粋関数、外部依存なし）

```go
// pkg/stages/layout/layout.go
type LayoutStage struct{}

func NewLayoutStage() *LayoutStage

func (s *LayoutStage) Execute(ctx context.Context, input LayoutInput) (LayoutResult, error)

// 内部メソッド
func (s *LayoutStage) calculateColumns(input LayoutInput) []Rectangle
func (s *LayoutStage) calculateWindows(columns []Rectangle, scrollHeight int) []Window
func (s *LayoutStage) calculateViewportSize(input LayoutInput) Dimension
```

**テスト**: モック不要、入力→出力の検証のみ

### 2. Record Stage

**責務**: ブラウザ録画

```go
// pkg/stages/record/record.go
type RecordStage struct {
    browser Browser
    sink    DebugSink
}

func NewRecordStage(browser Browser, sink DebugSink) *RecordStage

func (s *RecordStage) Execute(ctx context.Context, input RecordInput) (RecordResult, error)
```

**テスト**: MockBrowser を注入

### 3. Banner Stage

**責務**: バナー画像生成（Pure Go）

```go
// pkg/stages/banner/banner.go
type BannerStage struct {
    renderer Renderer
    sink     DebugSink
}

func NewBannerStage(renderer Renderer, sink DebugSink) *BannerStage

func (s *BannerStage) Execute(ctx context.Context, input BannerInput) (BannerResult, error)
```

**テスト**: MockRenderer を注入

### 4. Composite Stage

**責務**: フレーム合成（並列処理対応）

```go
// pkg/stages/composite/composite.go
type CompositeStage struct {
    renderer   Renderer
    sink       DebugSink
    numWorkers int
}

func NewCompositeStage(renderer Renderer, sink DebugSink, numWorkers int) *CompositeStage

func (s *CompositeStage) Execute(ctx context.Context, input CompositeInput) (CompositeResult, error)

// 単一フレームの合成（並列実行される）
func (s *CompositeStage) composeFrame(input CompositeInput, frameIndex int) (ComposedFrame, error)
```

**並列処理**:

```go
// pkg/stages/composite/parallel.go
func (s *CompositeStage) executeParallel(ctx context.Context, input CompositeInput) (CompositeResult, error) {
    jobs := make(chan int, len(input.RawFrames))
    results := make(chan indexedFrame, len(input.RawFrames))
    errs := make(chan error, s.numWorkers)

    // ワーカー起動
    var wg sync.WaitGroup
    for w := 0; w < s.numWorkers; w++ {
        wg.Add(1)
        go s.worker(ctx, &wg, input, jobs, results, errs)
    }

    // ジョブ投入
    for i := range input.RawFrames {
        jobs <- i
    }
    close(jobs)

    // 完了待ち
    go func() {
        wg.Wait()
        close(results)
        close(errs)
    }()

    // エラーチェック
    if err := <-errs; err != nil {
        return CompositeResult{}, err
    }

    // 結果収集・ソート
    frames := s.collectAndSort(results)
    return CompositeResult{Frames: frames}, nil
}
```

**テスト**: MockRenderer を注入

### 5. Encode Stage

**責務**: WebM エンコーディング

```go
// pkg/stages/encode/encode.go
type EncodeStage struct {
    encoder VideoEncoder
}

func NewEncodeStage(encoder VideoEncoder) *EncodeStage

func (s *EncodeStage) Execute(ctx context.Context, input EncodeInput) (EncodeResult, error)
```

**テスト**: MockVideoEncoder を注入

## Orchestrator

**責務**: 全ステージの実行順序とデータフローを制御

```go
// pkg/orchestrator/orchestrator.go
type Orchestrator struct {
    layoutStage    Stage[LayoutInput, LayoutResult]
    recordStage    Stage[RecordInput, RecordResult]
    bannerStage    Stage[BannerInput, BannerResult]
    compositeStage Stage[CompositeInput, CompositeResult]
    encodeStage    Stage[EncodeInput, EncodeResult]
    fs             FileSystem
    sink           DebugSink
}

type OrchestratorConfig struct {
    // 入力
    URL        string
    OutputPath string

    // レイアウト
    CanvasWidth    int
    CanvasHeight   int
    Columns        int
    Gap            int
    Padding        int

    // 録画
    ViewportWidth     int
    TimeoutMs         int
    NetworkConditions NetworkConditions
    CPUThrottling     float64

    // バナー
    BannerEnabled bool
    BannerHeight  int

    // エンコード
    Codec   VPXCodec
    Quality int
    Bitrate int
    OutroMs int
}

func NewOrchestrator(
    layoutStage Stage[LayoutInput, LayoutResult],
    recordStage Stage[RecordInput, RecordResult],
    bannerStage Stage[BannerInput, BannerResult],
    compositeStage Stage[CompositeInput, CompositeResult],
    encodeStage Stage[EncodeInput, EncodeResult],
    fs FileSystem,
    sink DebugSink,
) *Orchestrator

func (o *Orchestrator) Run(ctx context.Context, config OrchestratorConfig) error {
    // 1. レイアウト計算
    layoutInput := o.buildLayoutInput(config)
    layout, err := o.layoutStage.Execute(ctx, layoutInput)
    if err != nil {
        return fmt.Errorf("layout stage: %w", err)
    }
    o.sink.SaveLayout(layout)

    // 2. ブラウザ録画
    recordInput := o.buildRecordInput(config, layout)
    record, err := o.recordStage.Execute(ctx, recordInput)
    if err != nil {
        return fmt.Errorf("record stage: %w", err)
    }

    // 3. バナー生成（オプション）
    var banner *BannerResult
    if config.BannerEnabled {
        bannerInput := o.buildBannerInput(config, record)
        b, err := o.bannerStage.Execute(ctx, bannerInput)
        if err != nil {
            return fmt.Errorf("banner stage: %w", err)
        }
        banner = &b
    }

    // 4. フレーム合成
    compositeInput := o.buildCompositeInput(config, layout, record, banner)
    composite, err := o.compositeStage.Execute(ctx, compositeInput)
    if err != nil {
        return fmt.Errorf("composite stage: %w", err)
    }

    // 5. WebMエンコード
    encodeInput := o.buildEncodeInput(config, composite)
    encoded, err := o.encodeStage.Execute(ctx, encodeInput)
    if err != nil {
        return fmt.Errorf("encode stage: %w", err)
    }

    // 6. ファイル出力
    if err := o.fs.WriteFile(config.OutputPath, encoded.WebMData); err != nil {
        return fmt.Errorf("write output: %w", err)
    }

    return nil
}
```

## Adapters（実装）

### ChromeBrowser

```go
// pkg/adapters/chromebrowser/browser.go
type ChromeBrowser struct {
    ctx      context.Context
    cancel   context.CancelFunc
    allocCtx context.Context
}

func NewChromeBrowser() *ChromeBrowser
// Browser インターフェースを実装
```

### GGRenderer

```go
// pkg/adapters/ggrenderer/renderer.go
type GGRenderer struct{}

func NewGGRenderer() *GGRenderer
// Renderer インターフェースを実装

type GGCanvas struct {
    dc *gg.Context
}
// Canvas インターフェースを実装
```

### VPXEncoder

```go
// pkg/adapters/vpxencoder/encoder.go
// #cgo pkg-config: vpx
// #include <vpx/vpx_encoder.h>
// #include <vpx/vp8cx.h>
import "C"

type VPXEncoder struct {
    codec   *C.vpx_codec_ctx_t
    cfg     *C.vpx_codec_enc_cfg_t
    frames  [][]byte
    width   int
    height  int
}

func NewVPXEncoder() *VPXEncoder
// VideoEncoder インターフェースを実装
```

### OSFileSystem

```go
// pkg/adapters/osfilesystem/filesystem.go
type OSFileSystem struct{}

func NewOSFileSystem() *OSFileSystem
// FileSystem インターフェースを実装
```

### FileSink / NullSink

```go
// pkg/adapters/filesink/sink.go
type FileSink struct {
    baseDir  string
    fs       FileSystem
    renderer Renderer
}

func NewFileSink(baseDir string, fs FileSystem, renderer Renderer) *FileSink
// DebugSink インターフェースを実装

// pkg/adapters/nullsink/sink.go
type NullSink struct{}

func NewNullSink() *NullSink
func (s *NullSink) Enabled() bool { return false }
// 他のメソッドは何もしない
```

## Mocks（テスト用）

```go
// pkg/mocks/browser.go
type MockBrowser struct {
    LaunchFunc        func(ctx context.Context, opts BrowserOptions) error
    NavigateFunc      func(url string) error
    StartScreencastFunc func(quality int) (<-chan ScreenFrame, error)
    // ...
}

// pkg/mocks/renderer.go
type MockRenderer struct {
    CreateCanvasFunc func(width, height int, bg color.Color) Canvas
    DecodeImageFunc  func(data []byte, format ImageFormat) (image.Image, error)
    // ...
}

// pkg/mocks/encoder.go
type MockVideoEncoder struct {
    BeginFunc       func(width, height int, fps float64, opts EncoderOptions) error
    EncodeFrameFunc func(img image.Image, timestampMs int) error
    EndFunc         func() ([]byte, error)
}

// pkg/mocks/filesystem.go
type MockFileSystem struct {
    files map[string][]byte
}

// pkg/mocks/sink.go
type MockDebugSink struct {
    layouts  []LayoutResult
    frames   map[int]RawFrame
    // ...
}
```

## プロジェクト構造

```text
loadshow/
├── cmd/
│   └── loadshow/
│       └── main.go
├── pkg/
│   ├── ports/                    # インターフェース定義
│   │   ├── browser.go
│   │   ├── renderer.go
│   │   ├── encoder.go
│   │   ├── filesystem.go
│   │   └── sink.go
│   ├── adapters/                 # 実装
│   │   ├── chromebrowser/
│   │   │   └── browser.go
│   │   ├── ggrenderer/
│   │   │   └── renderer.go
│   │   ├── vpxencoder/
│   │   │   └── encoder.go
│   │   ├── osfilesystem/
│   │   │   └── filesystem.go
│   │   ├── filesink/
│   │   │   └── sink.go
│   │   └── nullsink/
│   │       └── sink.go
│   ├── pipeline/                 # パイプライン基盤
│   │   ├── stage.go
│   │   └── types.go
│   ├── stages/                   # 各ステージ
│   │   ├── layout/
│   │   │   ├── layout.go
│   │   │   └── layout_test.go
│   │   ├── record/
│   │   │   ├── record.go
│   │   │   └── record_test.go
│   │   ├── banner/
│   │   │   ├── banner.go
│   │   │   └── banner_test.go
│   │   ├── composite/
│   │   │   ├── composite.go
│   │   │   ├── parallel.go
│   │   │   └── composite_test.go
│   │   └── encode/
│   │       ├── encode.go
│   │       └── encode_test.go
│   ├── orchestrator/
│   │   ├── orchestrator.go
│   │   └── orchestrator_test.go
│   ├── mocks/
│   │   ├── browser.go
│   │   ├── renderer.go
│   │   ├── encoder.go
│   │   ├── filesystem.go
│   │   └── sink.go
│   └── config/
│       └── config.go
├── go.mod
├── go.sum
└── Makefile
```

## テスト戦略

### 単体テスト（各ステージ）

```go
// pkg/stages/layout/layout_test.go
func TestLayoutStage_Execute(t *testing.T) {
    stage := NewLayoutStage()

    input := LayoutInput{
        CanvasWidth:  512,
        CanvasHeight: 640,
        Columns:      3,
        Gap:          20,
        Padding:      20,
    }

    result, err := stage.Execute(context.Background(), input)

    assert.NoError(t, err)
    assert.Equal(t, 3, len(result.Columns))
    assert.Equal(t, 3, len(result.Windows))
    // 各カラムの位置・サイズを検証
}

// pkg/stages/composite/composite_test.go
func TestCompositeStage_Execute(t *testing.T) {
    mockRenderer := &mocks.MockRenderer{
        CreateCanvasFunc: func(w, h int, bg color.Color) Canvas {
            return &mocks.MockCanvas{}
        },
        DecodeImageFunc: func(data []byte, format ImageFormat) (image.Image, error) {
            return image.NewRGBA(image.Rect(0, 0, 100, 100)), nil
        },
    }
    mockSink := mocks.NewMockDebugSink()

    stage := NewCompositeStage(mockRenderer, mockSink, 4)

    input := CompositeInput{
        RawFrames: []RawFrame{
            {TimestampMs: 0, ImageData: []byte{...}},
            {TimestampMs: 100, ImageData: []byte{...}},
        },
        Layout: testLayout,
    }

    result, err := stage.Execute(context.Background(), input)

    assert.NoError(t, err)
    assert.Equal(t, 2, len(result.Frames))
    assert.Equal(t, 0, result.Frames[0].TimestampMs)
    assert.Equal(t, 100, result.Frames[1].TimestampMs)
}
```

### 統合テスト（Orchestrator）

```go
// pkg/orchestrator/orchestrator_test.go
func TestOrchestrator_Run(t *testing.T) {
    // モックステージを作成
    mockLayout := &MockLayoutStage{
        result: LayoutResult{...},
    }
    mockRecord := &MockRecordStage{
        result: RecordResult{
            Frames: []RawFrame{{TimestampMs: 0}},
        },
    }
    mockBanner := &MockBannerStage{
        result: BannerResult{},
    }
    mockComposite := &MockCompositeStage{
        result: CompositeResult{
            Frames: []ComposedFrame{{TimestampMs: 0}},
        },
    }
    mockEncode := &MockEncodeStage{
        result: EncodeResult{
            WebMData: []byte{0x1A, 0x45, 0xDF, 0xA3}, // WebM magic
        },
    }
    mockFS := mocks.NewMockFileSystem()
    mockSink := mocks.NewMockDebugSink()

    orch := NewOrchestrator(
        mockLayout, mockRecord, mockBanner,
        mockComposite, mockEncode, mockFS, mockSink,
    )

    err := orch.Run(context.Background(), OrchestratorConfig{
        URL:        "https://example.com",
        OutputPath: "output.webm",
    })

    assert.NoError(t, err)
    assert.True(t, mockFS.Exists("output.webm"))
}
```

## CLI コマンド

```bash
# 基本
loadshow https://example.com -o output.webm

# オプション指定
loadshow https://example.com -o output.webm \
    --columns 3 \
    --canvas-width 512 \
    --canvas-height 640 \
    --codec vp9 \
    --quality 30 \
    --banner

# デバッグモード（中間ファイル出力）
loadshow https://example.com -o output.webm --debug --debug-dir ./debug

# 設定ファイル
loadshow https://example.com -o output.webm --config config.yaml
```

## デバッグ出力

`--debug` フラグで中間ファイルを出力:

```text
debug/
├── layout.json           # レイアウト計算結果
├── layout.svg            # レイアウトのビジュアル表示
├── recording.json        # 録画メタデータ
├── banner.png            # 生成されたバナー
├── frames/
│   ├── raw/              # 録画された生フレーム
│   │   ├── frame-0001.jpg
│   │   └── ...
│   └── composed/         # 合成済みフレーム
│       ├── frame-0001.png
│       └── ...
└── output.webm           # 最終出力
```

## ビルド要件

### 依存ライブラリ

```bash
# macOS
brew install libvpx

# Ubuntu/Debian
apt-get install libvpx-dev

# Windows (MSYS2)
pacman -S mingw-w64-x86_64-libvpx
```

### go.mod

```go
module github.com/your/loadshow

go 1.21

require (
    github.com/alecthomas/kong v0.8.1
    github.com/chromedp/chromedp v0.9.3
    github.com/fogleman/gg v1.3.0
    golang.org/x/image v0.14.0
    gopkg.in/yaml.v3 v3.0.1
)
```

## 実装順序

1. **pkg/ports**: インターフェース定義
2. **pkg/pipeline/types.go**: 共通型定義
3. **pkg/stages/layout**: 外部依存なし、純粋関数
4. **pkg/mocks**: テスト用モック
5. **pkg/stages/record**: Browser依存、テスト作成
6. **pkg/stages/banner**: Renderer依存、テスト作成
7. **pkg/stages/composite**: Renderer依存、並列処理、テスト作成
8. **pkg/stages/encode**: VideoEncoder依存、テスト作成
9. **pkg/adapters**: 各実装
10. **pkg/orchestrator**: 全体統合、テスト作成
11. **cmd/loadshow**: CLI

## まとめ

この設計により、Loadshow の Go 実装は以下の特徴を持ちます：

1. **テスタブル**: 各ステージをモック可能、入出力を型で保証
2. **外部依存の分離**: Ports/Adapters パターンで抽象化
3. **パイプライン**: 各段階が独立、前工程→後工程のデータフローが明確
4. **並列処理**: CompositeStage でワーカープールによる高速化
5. **デバッグ支援**: DebugSink で中間ファイル出力
6. **ffmpeg 不要**: libvpx を CGO で直接利用、WebM 形式で出力
