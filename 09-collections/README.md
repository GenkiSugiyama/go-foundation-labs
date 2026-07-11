# 09-collections

## 目的
代表的なデータ構造をGoで実装し、操作と時間・空間計算量をテストやベンチマークで観察します。

## この課題で学ぶこと
- Stack、Queue、Set、LinkedList
- BinarySearch、Trie
- 時間計算量と空間計算量
- Go genericsの前提と型制約

## 完成イメージ
```go
var s Stack[int]
s.Push(10); s.Push(20)
v, _ := s.Pop()
fmt.Println(v) // 20
```

## ディレクトリ構成
```text
09-collections/
├── README.md
├── go.mod
├── stack.go / stack_test.go
├── queue.go / queue_test.go
├── set.go / set_test.go
├── linkedlist.go / linkedlist_test.go
├── search.go / search_test.go
├── trie.go / trie_test.go
└── benchmark_test.go
```

## 前提
Go 1.18以降が必要です。`go version`で確認し、Genericsを使わない場合はint/string版から始めます。
```bash
go version
go mod init example.com/go-foundation-labs/09-collections
```

## Step 1: StackとQueue
StackはLIFO、QueueはFIFOです。最初はスライスでPush/Pop/Enqueue/Dequeueを作り、空の場合もテストします。
```go
type Stack[T any] struct{ items []T }
func (s *Stack[T]) Push(v T) { s.items = append(s.items, v) }
func (s *Stack[T]) Pop() (T, bool) {
 var zero T
 if len(s.items) == 0 { return zero, false }
 i := len(s.items)-1; v := s.items[i]; s.items = s.items[:i]
 return v, true
}
```
末尾操作は償却O(1)です。Queueの先頭削除は保持領域にも注意します。

## Step 2: SetとLinkedList
```go
type Set[T comparable] map[T]struct{}

func NewSet[T comparable]() Set[T] {
 return make(Set[T])
}

func (s Set[T]) Add(v T) { s[v] = struct{}{} }
func (s Set[T]) Has(v T) bool { _, ok := s[v]; return ok }
```
`Set` は `NewSet` で初期化してから使います。nil mapへの `Add` はpanicになるため、ゼロ値で使える `Stack` との違いも確認してください。Setの平均追加・検索はO(1)、LinkedListの探索はO(n)、先頭追加はO(1)です。map、スライス、ノードの空間使用量を比較します。

## Step 3: BinarySearchとTrie
ソート済みスライスを二分するBinarySearchを作ります。
```go
func BinarySearch(xs []int, target int) int {
 lo, hi := 0, len(xs)-1
 for lo <= hi {
  mid := lo + (hi-lo)/2
  if xs[mid] == target { return mid }
  if xs[mid] < target { lo = mid+1 } else { hi = mid-1 }
 }
 return -1
}
```
BinarySearchは前提がソート済みで、計算量はO(log n)です。Trieはmap[rune]*nodeの子を持ち、単語長Lに対してInsertとprefix検索が概ねO(L)です。

## Step 4: テストとベンチマーク
```go
func BenchmarkBinarySearch(b *testing.B) {
 values := make([]int, 1<<20)
 for i := range values { values[i] = i }
 b.ResetTimer()
 for i := 0; i < b.N; i++ { _ = BinarySearch(values, len(values)-1) }
}
```
```bash
go test ./...
go test -bench=BinarySearch -benchmem ./...
```
数値は環境で変わるため、入力を増やした傾向とループ構造を合わせて説明します。

## ここで確認すること
- 各構造の代表操作と計算量
- 空の取り出し、未存在検索、重複追加の扱い
- BinarySearchがソート済み入力を要求すること
- Trieがprefix検索に有利な代わりに空間を使うこと

## 詰まったこと
学習後に追記します。
- 境界条件、ポインタ、型制約のどこで詰まりましたか。
- テストやベンチマークで何を観察しましたか。

## 分かったこと
学習後に追記します。
- 各構造をどの操作のために選びますか。
- 時間計算量と空間計算量のトレードオフは何ですか。

## 実務での関係
キャッシュ、キュー、重複排除、検索、補完などの性能と実装選択に関係します。Big Oだけでなく入力分布とメモリも確認します。

## 理解度チェック
1. StackとQueueの取り出し順は何が違いますか。
2. BinarySearchの前提と計算量は何ですか。
3. Trieの計算量を単語長で表す理由は何ですか。

## 発展課題
- Queueをリングバッファにする
- 比較関数を受け取るBinarySearchを作る
- Setの和・積・差、Trieの削除を実装する

## 覚えるべきキーワード
Stack、Queue、Set、LinkedList、BinarySearch、Trie、LIFO、FIFO、Big O、時間計算量、空間計算量、generics、benchmark。

## この課題のゴール
要件、計算量、メモリ、入力の前提を比較してデータ構造を選び、その理由を説明できることです。
