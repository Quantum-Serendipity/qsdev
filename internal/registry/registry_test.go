package registry

import (
	"fmt"
	"sort"
	"sync"
	"testing"
)

func TestNew_DefaultConfig(t *testing.T) {
	t.Parallel()
	r := New[string]()
	if r.Count() != 0 {
		t.Fatalf("Count() = %d, want 0", r.Count())
	}
}

func TestRegister_DenyDuplicates(t *testing.T) {
	t.Parallel()
	r := New[int](WithEntityName("widget"))

	if err := r.Register("a", 1); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	err := r.Register("a", 2)
	if err == nil {
		t.Fatal("expected error for duplicate key")
	}
	want := `widget "a" already registered`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
	// Original value is preserved.
	got, _ := r.Get("a")
	if got != 1 {
		t.Errorf("Get(a) = %d, want 1 (original)", got)
	}
}

func TestRegister_AllowOverwrite(t *testing.T) {
	t.Parallel()
	r := New[string](WithDuplicatePolicy(AllowOverwrite))

	_ = r.Register("key", "v1")
	if err := r.Register("key", "v2"); err != nil {
		t.Fatalf("overwrite Register: %v", err)
	}
	got, ok := r.Get("key")
	if !ok || got != "v2" {
		t.Errorf("Get(key) = (%q, %v), want (v2, true)", got, ok)
	}
	if r.Count() != 1 {
		t.Errorf("Count() = %d, want 1 after overwrite", r.Count())
	}
}

func TestGet_NotFound(t *testing.T) {
	t.Parallel()
	r := New[int]()
	_, ok := r.Get("missing")
	if ok {
		t.Error("Get(missing) returned true")
	}
}

func TestNames_SortedByDefault(t *testing.T) {
	t.Parallel()
	r := New[int]()
	_ = r.Register("charlie", 3)
	_ = r.Register("alpha", 1)
	_ = r.Register("bravo", 2)

	names := r.Names()
	want := []string{"alpha", "bravo", "charlie"}
	if len(names) != len(want) {
		t.Fatalf("Names() len = %d, want %d", len(names), len(want))
	}
	for i, n := range names {
		if n != want[i] {
			t.Errorf("Names()[%d] = %q, want %q", i, n, want[i])
		}
	}
}

func TestNames_InsertionOrder(t *testing.T) {
	t.Parallel()
	r := New[int](WithInsertionOrder())
	_ = r.Register("charlie", 3)
	_ = r.Register("alpha", 1)
	_ = r.Register("bravo", 2)

	names := r.Names()
	want := []string{"charlie", "alpha", "bravo"}
	if len(names) != len(want) {
		t.Fatalf("Names() len = %d, want %d", len(names), len(want))
	}
	for i, n := range names {
		if n != want[i] {
			t.Errorf("Names()[%d] = %q, want %q", i, n, want[i])
		}
	}
}

func TestAll_ReturnsShallowCopy(t *testing.T) {
	t.Parallel()
	r := New[int]()
	_ = r.Register("x", 42)

	m := r.All()
	m["y"] = 99 // mutate returned map
	if r.Count() != 1 {
		t.Error("mutating All() result affected registry")
	}
}

func TestCount(t *testing.T) {
	t.Parallel()
	r := New[string]()
	if r.Count() != 0 {
		t.Fatalf("empty Count() = %d", r.Count())
	}
	_ = r.Register("a", "A")
	_ = r.Register("b", "B")
	if r.Count() != 2 {
		t.Fatalf("Count() = %d, want 2", r.Count())
	}
}

func TestRange(t *testing.T) {
	t.Parallel()
	r := New[int]()
	_ = r.Register("a", 1)
	_ = r.Register("b", 2)
	_ = r.Register("c", 3)

	var sum int
	r.Range(func(_ string, v int) bool {
		sum += v
		return true
	})
	if sum != 6 {
		t.Errorf("sum = %d, want 6", sum)
	}
}

func TestRange_EarlyStop(t *testing.T) {
	t.Parallel()
	r := New[int]()
	_ = r.Register("a", 1)
	_ = r.Register("b", 2)
	_ = r.Register("c", 3)

	var count int
	r.Range(func(_ string, _ int) bool {
		count++
		return false // stop after first
	})
	if count != 1 {
		t.Errorf("count = %d, want 1 (early stop)", count)
	}
}

func TestConcurrentAccess(t *testing.T) {
	t.Parallel()
	r := New[int]()

	var wg sync.WaitGroup
	// Concurrent writers.
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = r.Register(fmt.Sprintf("key-%03d", n), n)
		}(i)
	}
	// Concurrent readers.
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.Names()
			_ = r.Count()
			_ = r.All()
			_, _ = r.Get("key-000")
		}()
	}
	wg.Wait()

	if r.Count() != 100 {
		t.Errorf("Count() = %d, want 100", r.Count())
	}
	names := r.Names()
	if !sort.StringsAreSorted(names) {
		t.Error("Names() not sorted after concurrent access")
	}
}

func TestInsertionOrder_OverwriteDoesNotDuplicate(t *testing.T) {
	t.Parallel()
	r := New[string](WithInsertionOrder(), WithDuplicatePolicy(AllowOverwrite))

	_ = r.Register("a", "v1")
	_ = r.Register("b", "v2")
	_ = r.Register("a", "v3") // overwrite

	names := r.Names()
	want := []string{"a", "b"}
	if len(names) != len(want) {
		t.Fatalf("Names() len = %d, want %d", len(names), len(want))
	}
	for i, n := range names {
		if n != want[i] {
			t.Errorf("Names()[%d] = %q, want %q", i, n, want[i])
		}
	}

	got, _ := r.Get("a")
	if got != "v3" {
		t.Errorf("Get(a) = %q, want v3", got)
	}
}

func TestEntityName_DefaultIsItem(t *testing.T) {
	t.Parallel()
	r := New[int]()
	_ = r.Register("k", 1)
	err := r.Register("k", 2)
	if err == nil {
		t.Fatal("expected error")
	}
	want := `item "k" already registered`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestModify_Found(t *testing.T) {
	t.Parallel()
	type box struct{ val int }
	r := New[*box]()
	_ = r.Register("k", &box{val: 1})

	ok := r.Modify("k", func(b *box) { b.val = 42 })
	if !ok {
		t.Fatal("Modify returned false for existing key")
	}
	got, _ := r.Get("k")
	if got.val != 42 {
		t.Errorf("val = %d, want 42", got.val)
	}
}

func TestModify_NotFound(t *testing.T) {
	t.Parallel()
	r := New[int]()
	ok := r.Modify("missing", func(_ int) {})
	if ok {
		t.Error("Modify returned true for missing key")
	}
}

func TestDelete_Exists(t *testing.T) {
	t.Parallel()
	r := New[int]()
	_ = r.Register("a", 1)
	_ = r.Register("b", 2)

	ok := r.Delete("a")
	if !ok {
		t.Fatal("Delete returned false for existing key")
	}
	if r.Count() != 1 {
		t.Errorf("Count() = %d, want 1", r.Count())
	}
	_, found := r.Get("a")
	if found {
		t.Error("Get(a) returned true after Delete")
	}
}

func TestDelete_NotExists(t *testing.T) {
	t.Parallel()
	r := New[int]()
	ok := r.Delete("missing")
	if ok {
		t.Error("Delete returned true for missing key")
	}
}

func TestDelete_InsertionOrder(t *testing.T) {
	t.Parallel()
	r := New[int](WithInsertionOrder())
	_ = r.Register("a", 1)
	_ = r.Register("b", 2)
	_ = r.Register("c", 3)

	r.Delete("b")
	names := r.Names()
	want := []string{"a", "c"}
	if len(names) != len(want) {
		t.Fatalf("Names() len = %d, want %d", len(names), len(want))
	}
	for i, n := range names {
		if n != want[i] {
			t.Errorf("Names()[%d] = %q, want %q", i, n, want[i])
		}
	}
}
