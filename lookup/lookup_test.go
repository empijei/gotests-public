package lookup

import (
	"fmt"
	"sort"
	"strconv"
	"testing"
)

func TestSlice(t *testing.T) {
	hayStack := []int{8, 9, 1, 2, 3, 4, 7}
	if got, want := sliceHas(hayStack, 5), false; got != want {
		t.Errorf("sliceHas(%v, 5): got %v want %v", hayStack, got, want)
	}
	if got, want := sliceHas(hayStack, 7), true; got != want {
		t.Errorf("sliceHas(%v, 7): got %v want %v", hayStack, got, want)
	}

	_, hayStack = setupInt(5)
	if got, want := sliceHas(hayStack, 25), false; got != want {
		t.Errorf("sliceHas(%v, 25): got %v want %v", hayStack, got, want)
	}
	if got, want := sliceHas(hayStack, 2), true; got != want {
		t.Errorf("sliceHas(%v, 2): got %v want %v", hayStack, got, want)
	}
}

var sizes = []int{2, 4, 8, 16, 32, 64, 128}

func BenchmarkLargeData(b *testing.B) {
	for _, size := range sizes {
		b.Run(fmt.Sprintf("slice-%v", size), func(b *testing.B) {
			s := setupLargeSlice(size)
			var needle largeData
			for j := range needle {
				needle[j] = size / 2
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				sliceHas(s, needle)
			}
		})
		b.Run(fmt.Sprintf("map-%v", size), func(b *testing.B) {
			s := setupLargeMap(size)
			var needle largeData
			for j := range needle {
				needle[j] = size / 2
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				mapHas(s, needle)
			}
		})
	}
}
func BenchmarkInts(b *testing.B) {
	for _, size := range sizes {
		b.Run(fmt.Sprintf("slice-%v", size), func(b *testing.B) {
			s := setupIntSlice(size)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				sliceHas(s, size/2)
			}
		})
		b.Run(fmt.Sprintf("map-%v", size), func(b *testing.B) {
			s := setupIntMap(size)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				mapHas(s, size/2)
			}
		})
	}
}
func BenchmarkStrings(b *testing.B) {
	for _, size := range sizes {
		b.Run(fmt.Sprintf("slice-%v", size), func(b *testing.B) {
			s := setupStringSlice(size)
			needle := strconv.Itoa(size / 2)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				sliceHas(s, needle)
			}
		})
		b.Run(fmt.Sprintf("map-%v", size), func(b *testing.B) {
			s := setupStringMap(size)
			needle := strconv.Itoa(size / 2)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				mapHas(s, needle)
			}
		})
	}
}

func TestCutoff(t *testing.T) {
	t.Skip("this doesn't work")
	// Make sure we have bounds for the search.
	const end = 128
	if got := isSliceFasterInt(end); got == true {
		t.Fatalf("Map of size %v is still slower than slice", end)
	}

	i, ok := sort.Find(end, cmpInt)
	if !ok {
		t.Fatalf("Cutoff not found")
	}
	fmt.Println(i)
}
