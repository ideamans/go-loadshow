# loadshow Recording におけるブラウザ設定

## 概要

loadshow（TypeScript版）のrecordingステージでは、Puppeteer経由でChromeを操作し、ページ読み込みをスクリーンキャストで記録します。

## 設定の階層

```
RecordingSpec (ユーザー設定)
    ↓
RecordingInput (内部処理用)
    ↓
Puppeteer Launch + CDP Session
```

## 1. Puppeteer Launch Options

### デフォルト設定 (`recording.ts:27-38`)

```typescript
const defaultPuppeteerLaunchOptions = {
  headless: 'new',      // 新しいheadlessモード
  args: ['--scrollbars'],
}

// puppeteer-core使用時
const defaultCorePuppeteerLaunchOptions = {
  headless: true,
  args: ['--scrollbars'],
}
```

### Chrome実行パスの解決順序 (`dependency.ts:79-109`)

1. `CHROME_PATH` 環境変数
2. システムインストール済みChrome（`preferSystemChrome: true`の場合）
3. バンドル版Chromium（`BARE_PUPPETEER=1`の場合のみ）

## 2. Viewport設定

### 計算式 (`recording.ts:150-158`)

```typescript
// deviceScaleFactorの計算
const deviceScaleFactor = input.screen.width / input.viewportWidth

// viewportの高さ計算（Linux用の補正付き）
const extraHeight = process.platform === 'linux' ? 1.1 : 1
const viewport = {
  width: input.viewportWidth,
  height: Math.ceil((input.screen.height / deviceScaleFactor) * extraHeight),
}

// viewportの設定
await page.setViewport({ ...viewport, deviceScaleFactor })
```

### 入力パラメータ

| パラメータ | 説明 | デフォルト |
|-----------|------|-----------|
| `viewportWidth` | CSSピクセル幅 | 375 |
| `screen.width` | 出力画像幅（ピクセル） | layout依存 |
| `screen.height` | 出力画像高さ（ピクセル） | layout依存 |

### 計算例

```
screen: 375 x 2000 (出力サイズ)
viewportWidth: 375

deviceScaleFactor = 375 / 375 = 1.0
viewport = { width: 375, height: 2000 }
```

```
screen: 750 x 4000 (2倍の出力サイズ)
viewportWidth: 375

deviceScaleFactor = 750 / 375 = 2.0
viewport = { width: 375, height: 2000 }
→ 375x2000 CSS pixels が 750x4000 device pixels として出力
```

## 3. Window Bounds設定（Linux対応）

Linux環境では、viewport設定後にwindow boundsを追加設定 (`recording.ts:184-188`)

```typescript
if (process.platform === 'linux') {
  const { windowId } = await cdp.send('Browser.getWindowForTarget')
  await cdp.send('Browser.setWindowBounds', { windowId, bounds: viewport })
}
```

## 4. Screencast設定

### 開始 (`recording.ts:299-304`)

```typescript
await cdp.send('Page.startScreencast', {
  format: 'jpeg',
  quality: input.frameQuality,
  everyNthFrame: 1,
})
```

### 注意点

- **maxWidth/maxHeightを指定していない**
- screencastはviewportサイズに従ってフレームを生成
- フォーマットはJPEG固定

### フレーム受信 (`recording.ts:279-296`)

```typescript
cdp.on('Page.screencastFrame', async (f) => {
  const time = Math.floor((f.metadata?.timestamp ?? 0) * 1000) - startedAt
  output.screenFrames.push({
    time,
    resourcesLoading: { images: 0, all: 0 },
    base64Data: f.data,
  })
  await cdp.send('Page.screencastFrameAck', { sessionId: f.sessionId })
})
```

## 5. CDP経由のその他設定

### ネットワーク条件 (`recording.ts:191-199`)

```typescript
await cdp.send('Network.enable')
await cdp.send('Network.setCacheDisabled', { cacheDisabled: true })
await cdp.send('Network.emulateNetworkConditions', {
  offline: false,
  latency: input.network.latencyMs,
  downloadThroughput: Math.floor((input.network.downloadThroughputMbps * 1024 * 1024) / 8),
  uploadThroughput: Math.floor((input.network.uploadThroughputMbps * 1024 * 1024) / 8),
})
```

### CPUスロットリング (`recording.ts:202-203`)

```typescript
await cdp.send('Emulation.setCPUThrottlingRate', { rate: input.cpuThrottling })
```

## 6. 設定フロー図

```
┌─────────────────────────────────────────────────────────────┐
│ 1. Puppeteer Launch                                         │
│    - headless: true/'new'                                   │
│    - executablePath: Chrome path                            │
│    - args: ['--scrollbars']                                 │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│ 2. page.setViewport()                                       │
│    - width: viewportWidth (e.g., 375)                       │
│    - height: screen.height / deviceScaleFactor              │
│    - deviceScaleFactor: screen.width / viewportWidth        │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│ 3. page.setExtraHTTPHeaders()                               │
│    - Custom headers (User-Agent, Accept-Language, etc.)     │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│ 4. CDP Session Setup                                        │
│    a. Browser.setWindowBounds (Linux only)                  │
│    b. Network.enable + emulateNetworkConditions             │
│    c. Emulation.setCPUThrottlingRate                        │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│ 5. Page.startScreencast                                     │
│    - format: 'jpeg'                                         │
│    - quality: frameQuality                                  │
│    - everyNthFrame: 1                                       │
│    - (maxWidth/maxHeight: 未指定)                           │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│ 6. page.goto(url) + Event Handling                          │
│    - Listen for screencastFrame events                      │
│    - Acknowledge each frame                                 │
└─────────────────────────────────────────────────────────────┘
```

## 7. Go版(chromedp)との比較

| 項目 | TypeScript版 (Puppeteer) | Go版 (chromedp) |
|------|--------------------------|-----------------|
| viewport設定 | `page.setViewport()` | `emulation.SetDeviceMetricsOverride()` |
| window設定 | Linux: `Browser.setWindowBounds` | `browser.SetWindowBounds` + WindowSize flag |
| deviceScaleFactor | 動的計算 | 固定値 (1.0) |
| screencast format | jpeg | jpeg/png |
| screencast maxWidth/maxHeight | 未指定 | 指定 |

### Go版での問題点

Go版では`emulation.SetDeviceMetricsOverride`と`page.StartScreencast`の`maxWidth/maxHeight`を指定しているが、ページコンテンツがロードされるとフレームサイズがコンテンツの高さに変化する現象が発生。

TypeScript版では`maxWidth/maxHeight`を指定せず、`page.setViewport()`のみでサイズを制御している。

## 8. RecordingSpec デフォルト値

```typescript
{
  network: {
    latencyMs: 20,
    downloadThroughputMbps: 10,
    uploadThroughputMbps: 10,
  },
  cpuThrottling: 4,
  headers: {},
  viewportWidth: 375,
  timeoutMs: 30000,
  preferSystemChrome: false,
  puppeteer: {
    headless: true,
    args: ['--scrollbars'],
  },
}
```
