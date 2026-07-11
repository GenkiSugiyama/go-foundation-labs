# 08-notifier

## 目的
通知処理を小さなGoライブラリとして作り、interfaceと合成による設計を体験します。

## この課題で学ぶこと
- interface、Strategy、Observer、LSP
- composition over inheritance
- Logging/RetryをDecorator的に重ねる構成
- 外部サービスなしでテストできる設計

## 完成イメージ
```go
var n Notifier = ConsoleNotifier{}
n = LoggingNotifier{Next: n, Log: log.Default()}
n = RetryNotifier{Next: n, Attempts: 3}
_ = n.Notify(context.Background(), "build finished")
```

## ディレクトリ構成
```text
08-notifier/
├── README.md
├── go.mod
├── notifier.go
├── notifier_test.go
└── cmd/demo/main.go
```

## Step 1: ConsoleNotifierを作る
```bash
go mod init example.com/go-foundation-labs/08-notifier
mkdir -p cmd/demo
```
```go
type Notifier interface {
 Notify(context.Context, string) error
}
type ConsoleNotifier struct{}
func (ConsoleNotifier) Notify(_ context.Context, message string) error {
 _, err := fmt.Println(message)
 return err
}
```
`go run ./cmd/demo`で出力を観察します。interfaceには通知先に必要な契約だけを置きます。

## Step 2: StrategyとObserverを観察する
同じNotifyを持つMemoryNotifierを作り、テストで差し替えます。
```go
type MemoryNotifier struct{ Messages []string }
func (m *MemoryNotifier) Notify(_ context.Context, message string) error {
 m.Messages = append(m.Messages, message)
 return nil
}
type MultiNotifier []Notifier
func (m MultiNotifier) Notify(ctx context.Context, message string) error {
 for _, n := range m {
  if err := n.Notify(ctx, message); err != nil { return err }
 }
 return nil
}
```
通知先の差し替えはStrategy、1イベントを複数へ届ける構成はObserver的な例です。ただし、この `MultiNotifier` は購読登録・解除を持たない固定的な一斉配信なので、完全なObserver実装ではありません。パターン名より、どの依存関係を変更しやすくしたいかを説明できることを優先します。

## Step 3: LoggingとRetryを合成する
```go
type LoggingNotifier struct {
 Next Notifier
 Log interface{ Printf(string, ...any) }
}
func (n LoggingNotifier) Notify(ctx context.Context, message string) error {
 n.Log.Printf("notify %q", message)
 return n.Next.Notify(ctx, message)
}
type RetryNotifier struct { Next Notifier; Attempts int }
func (n RetryNotifier) Notify(ctx context.Context, message string) error {
 var err error
 for i := 0; i < max(1, n.Attempts); i++ {
  if err = n.Next.Notify(ctx, message); err == nil { return nil }
 }
 return err
}
```
`Next Notifier`を持たせるDecorator的な合成です。順序を変え、ログと試行回数を比較します。

## Step 4: テストで観察する
```go
type flaky struct{ calls int }
func (f *flaky) Notify(context.Context, string) error {
 f.calls++; if f.calls < 3 { return errors.New("temporary") }; return nil
}
func TestRetry(t *testing.T) {
 f := &flaky{}
 if err := (RetryNotifier{Next:f, Attempts:3}).Notify(context.Background(), "x"); err != nil { t.Fatal(err) }
 if f.calls != 3 { t.Fatalf("calls=%d", f.calls) }
}
```
`go test -v ./...`で成功回数を確認します。LSPの観点から、MemoryNotifierをConsoleNotifierの代わりに使える契約か考えます。

## ここで確認すること
- 呼び出し側が具体型でなくNotifierに依存すること
- LoggingとRetryの順序で結果が変わること
- interfaceを増やしすぎていないこと

## 詰まったこと
学習後に追記します。
- どのinterfaceやテストで詰まりましたか。
- Retryの回数や失敗条件をどう切り分けましたか。

## 分かったこと
学習後に追記します。
- Strategy、Observer、Decorator的構成はどこに現れましたか。
- 継承より合成を選ぶと何が差し替えやすくなりましたか。

## 実務での関係
通知チャネル、ログ、リトライ、モックを分離すると、外部サービス変更や障害時のテストが容易になります。リトライは重複通知や非冪等処理にも注意が必要です。

## 理解度チェック
1. MemoryNotifierがテストしやすい理由は何ですか。
2. LSPの観点でNotifier実装に必要な契約は何ですか。
3. Retryを合成で作る利点は何ですか。

## 発展課題
- contextのキャンセルを検知する
- 複数通知のエラーを集約する
- 待機時間を注入してテストする

## 覚えるべきキーワード
interface、Strategy、Observer、composition over inheritance、LSP、Decorator、依存性逆転、モック。

## この課題のゴール
Console通知を起点にログ・リトライ・複数通知を合成し、テストで設計を説明できることです。
