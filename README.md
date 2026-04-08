# woodriver

Go で書かれた W3C WebDriver クライアントライブラリ。Chrome・Firefox などのブラウザを HTTP 経由で自動操作します。

## 必要環境

- Go 1.21 以上
- ChromeDriver または GeckoDriver（操作対象ブラウザのドライバー）

## インストール

```bash
go get github.com/frappuccino2316/woodriver
```

## クイックスタート

```go
package main

import (
    "fmt"
    "log"

    "github.com/frappuccino2316/woodriver"
)

func main() {
    // ChromeDriver が localhost:9515 で起動していること
    driver := woodriver.New("http://localhost:9515")

    sess, err := driver.NewSession(woodriver.HeadlessChrome())
    if err != nil {
        log.Fatal(err)
    }
    defer sess.Quit()

    sess.Navigate("https://example.com")

    title, _ := sess.Title()
    fmt.Println(title) // "Example Domain"
}
```

## ドライバーの起動

```bash
# Chrome
chromedriver --port=9515

# Firefox
geckodriver --port=4444
```

## セッションの作成

### ケイパビリティ（ブラウザ設定）

```go
// ヘッドレス Chrome（コンテナ向け設定済みプリセット）
caps := woodriver.HeadlessChrome()

// ヘッドレス Firefox
caps := woodriver.HeadlessFirefox()

// Chrome オプションを細かく指定
caps := woodriver.ChromeCapabilities(
    woodriver.Headless(),
    woodriver.WindowSize(1920, 1080),
    woodriver.NoSandbox(),
    woodriver.DisableGPU(),
    woodriver.DisableDevShmUsage(),
    woodriver.ChromePref("download.default_directory", "/tmp"),
)

// Firefox オプション
caps := woodriver.FirefoxCapabilities(
    woodriver.FirefoxHeadless(),
    woodriver.FirefoxPref("browser.startup.homepage", "about:blank"),
    woodriver.FirefoxBinary("/usr/bin/firefox"),
)

// プロキシ設定
caps := woodriver.ChromeCapabilities(
    woodriver.WithProxy(woodriver.ManualProxy("proxy.example.com:8080")),
)

// モバイルエミュレーション
caps := woodriver.ChromeCapabilities(
    woodriver.EmulateDevice(woodriver.MobileDevice{DeviceName: "iPhone 12"}),
)
```

### カスタム HTTP クライアント

```go
import "net/http"

driver := woodriver.New(
    "http://localhost:9515",
    woodriver.WithHTTPClient(&http.Client{Timeout: 60 * time.Second}),
)
```

## ナビゲーション

```go
sess.Navigate("https://example.com")

url, _   := sess.CurrentURL()
title, _ := sess.Title()

sess.Back()
sess.Forward()
sess.Refresh()
```

## 要素の検索

### セレクター

```go
// CSS セレクター
el, err := sess.FindElement(woodriver.ByCSSSelector, "h1")
el, err := sess.FindElement(woodriver.ByCSSSelector, ".btn-primary")

// ID・name 属性（CSS セレクターの糖衣構文）
by, value := woodriver.ByID("submit-btn")
el, err   := sess.FindElement(by, value)

by, value = woodriver.ByName("username")
el, err   = sess.FindElement(by, value)

// XPath
el, err := sess.FindElement(woodriver.ByXPath, "//button[@type='submit']")

// リンクテキスト
el, err := sess.FindElement(woodriver.ByLinkText, "詳細を見る")

// 複数要素の取得
items, err := sess.FindElements(woodriver.ByCSSSelector, "ul li")
```

### 要素の操作

```go
el.Click()
el.SendKeys("hello, world")
el.Clear()

text, _      := el.Text()
href, _      := el.Attribute("href")
value, _     := el.Property("value")
tag, _       := el.TagName()
rect, _      := el.Rect()
displayed, _ := el.IsDisplayed()
enabled, _   := el.IsEnabled()
selected, _  := el.IsSelected()

// 子要素の検索
child, _ := el.FindElement(woodriver.ByCSSSelector, "span")
```

## 明示的待機

```go
// 要素が DOM に現れるまで最大 10 秒待つ
el, err := sess.Wait(10 * time.Second).UntilElement(woodriver.ByCSSSelector, ".result")

// カスタム条件
err = sess.Wait(5 * time.Second).Until(woodriver.TitleContains("Dashboard"))
err = sess.Wait(5 * time.Second).Until(woodriver.URLMatches("/dashboard"))
err = sess.Wait(5 * time.Second).Until(woodriver.ElementVisible(woodriver.ByCSSSelector, "#modal"))
err = sess.Wait(5 * time.Second).Until(woodriver.ElementClickable(woodriver.ByCSSSelector, "#submit"))

// 独自条件の定義
err = sess.Wait(10 * time.Second).Until(func(s woodriver.Session) (bool, error) {
    count, err := s.Execute(`return document.querySelectorAll(".item").length`)
    if err != nil {
        return false, err
    }
    return count.(float64) >= 5, nil
})
```

## JavaScript 実行

```go
// 同期実行
result, err := sess.Execute(`return document.title`)

// 引数を渡す
result, err = sess.Execute(`return arguments[0] + arguments[1]`, 1, 2)

// 非同期実行（Promise）
result, err = sess.ExecuteAsync(`
    const [resolve] = arguments;
    setTimeout(() => resolve("done"), 1000);
`)
```

## ウィンドウ操作

```go
// サイズ・位置
sess.SetWindowRect(woodriver.Rect{X: 0, Y: 0, Width: 1280, Height: 800})
rect, _ := sess.WindowRect()

// 状態変更
sess.Maximize()
sess.Minimize()
sess.Fullscreen()

// 複数ウィンドウ・タブ
handle, _  := sess.CurrentWindowHandle()
handles, _ := sess.WindowHandles()
sess.SwitchToWindow(handles[1])

tab, _ := sess.NewWindow(woodriver.WindowTypeTab)
sess.SwitchToWindow(tab.Handle)

// 現在のウィンドウを閉じる（セッションは維持）
sess.Close()
```

## フレーム操作

```go
// インデックスで切り替え
sess.SwitchToFrame(0)

// Element で切り替え
frame, _ := sess.FindElement(woodriver.ByCSSSelector, "iframe#content")
sess.SwitchToFrame(frame)

// トップレベルに戻る
sess.SwitchToFrame(nil)
sess.SwitchToParentFrame()
```

## アラート・ダイアログ

```go
text, _ := sess.AlertText()
sess.AcceptAlert()
sess.DismissAlert()
sess.SendAlertText("入力テキスト")
```

## Actions API

マウス・キーボード・ホイールの複合操作を組み立てて一括送信します。

```go
// 要素をクリック
sess.Actions().ClickElement(el).Perform()

// マウス操作
sess.Actions().
    MouseMove(100, 200).        // 座標へ移動
    MouseClick(woodriver.MouseLeft). // クリック
    Perform()

// キーボードショートカット（Ctrl+A）
sess.Actions().
    KeyDown(woodriver.KeyControl).
    KeySendKeys("a").
    KeyUp(woodriver.KeyControl).
    Perform()

// テキスト入力（フォーカス後）
sess.Actions().
    ClickElement(input).
    KeySendKeys("hello").
    Perform()

// スクロール
sess.Actions().Scroll(0, 0, 0, 500).Perform() // 500px 下へ

// ドラッグ＆ドロップ
sess.Actions().
    MouseMoveToElement(src).
    MouseDown(woodriver.MouseLeft).
    MouseMoveToElement(dst).
    MouseUp(woodriver.MouseLeft).
    Perform()
```

### キー定数

```go
woodriver.KeyEnter      // Enter
woodriver.KeyTab        // Tab
woodriver.KeyEscape     // Escape
woodriver.KeyBackspace  // Backspace
woodriver.KeyControl    // Ctrl
woodriver.KeyShift      // Shift
woodriver.KeyAlt        // Alt
woodriver.KeyMeta       // Cmd / Win
woodriver.KeyArrowUp    // ↑
woodriver.KeyF1         // F1 〜 F12
```

## Cookie 管理

```go
// 全取得
cookies, err := sess.Cookies()

// 名前で取得
c, err := sess.Cookie("session_id")

// 追加
sess.AddCookie(woodriver.Cookie{
    Name:     "token",
    Value:    "abc123",
    Path:     "/",
    Secure:   true,
    HTTPOnly: true,
    Expiry:   time.Now().Add(24 * time.Hour),
})

// 削除
sess.DeleteCookie("token")
sess.DeleteAllCookies()
```

## スクリーンショット

```go
png, err := sess.Screenshot()
os.WriteFile("screenshot.png", png, 0o644)
```

## 並列処理（SessionPool）

複数の URL を並列でスクレイピングする場合などに使います。

```go
pool, err := woodriver.NewSessionPool(
    context.Background(),
    woodriver.New("http://localhost:9515"),
    4,                            // 同時セッション数
    woodriver.HeadlessChrome(),
)
if err != nil {
    log.Fatal(err)
}
defer pool.Close()

var wg sync.WaitGroup
for _, url := range urls {
    wg.Add(1)
    go func(u string) {
        defer wg.Done()

        sess, err := pool.Acquire(context.Background())
        if err != nil {
            return
        }
        defer pool.Release(sess)

        sess.Navigate(u)
        title, _ := sess.Title()
        fmt.Println(title)
    }(url)
}
wg.Wait()
```

## エラーハンドリング

```go
el, err := sess.FindElement(woodriver.ByCSSSelector, "#missing")
if errors.Is(err, woodriver.ErrNoSuchElement) {
    // 要素が見つからなかった
}

// 主なエラー
woodriver.ErrNoSuchElement          // 要素が見つからない
woodriver.ErrStaleElementReference  // 要素が DOM から消えた
woodriver.ErrElementNotInteractable // 要素を操作できない
woodriver.ErrTimeout                // タイムアウト
woodriver.ErrInvalidSelector        // セレクターが不正
```

## インターフェース階層

```
Session        — 基本操作（ナビゲーション・要素検索・JS・スクリーンショット）
  └── WindowOps  — Session を拡張（ウィンドウ・フレーム・アラート・Cookie・Actions）
```

`driver.NewSession()` は `WindowOps` を返すため、型アサションなしで全機能を利用できます。

## ディレクトリ構成

```
woodriver/
├── doc.go                    # パッケージドキュメント
├── errors.go                 # WebDriverError・エラー定数
├── types.go                  # By・Rect・MouseButton
├── keys.go                   # キー定数
├── capabilities.go           # Capabilities・Chrome/Firefox ビルダー
├── session.go                # Session インターフェース・Driver
├── element.go                # Element インターフェース
├── window.go                 # WindowOps インターフェース
├── actions.go                # Actions API
├── wait.go                   # Waiter・Condition
├── cookies.go                # Cookie 操作
├── parallel.go               # SessionPool
├── internal/
│   └── transport/            # HTTP 通信層（非公開）
└── examples/
    ├── basic/                # 基本操作
    ├── actions/              # マウス・キーボード操作
    ├── scraping/             # スクレイピング
    └── parallel/             # 並列処理
```

## ライセンス

MIT
