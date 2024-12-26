package lookup

import (
	"fmt"
	"strconv"
	"time"
)

type largeData [100]int

type none struct{}

func sliceHas[T comparable](s []T, target T) bool {
	for _, v := range s {
		if v == target {
			return true
		}
	}
	return false
}

func mapHas[T comparable](m map[T]none, target T) bool {
	_, ok := m[target]
	return ok
}

func setupLargeMap(size int) map[largeData]none {
	m := make(map[largeData]none, size)
	for i := range size {
		var l largeData
		for j := range l {
			l[j] = i
		}
		m[l] = none{}
	}
	return m
}

func setupLargeSlice(size int) []largeData {
	s := make([]largeData, 0, size)
	for i := range size {
		var l largeData
		for j := range l {
			l[j] = i
		}
		s = append(s, l)
	}
	return s
}

func setupIntMap(size int) map[int]none {
	m := make(map[int]none, size)
	for i := range size {
		m[i] = none{}
	}
	return m
}

func setupIntSlice(size int) []int {
	s := make([]int, 0, size)
	for i := range size {
		s = append(s, i)
	}
	return s
}

func setupStringMap(size int) map[string]none {
	m := make(map[string]none, size)
	for i := range size {
		m[strconv.Itoa(i)] = none{}
	}
	return m
}

func setupStringSlice(size int) []string {
	s := make([]string, 0, size)
	for i := range size {
		s = append(s, strconv.Itoa(i))
	}
	return s
}

func setupInt(size int) (map[int]none, []int) {
	return setupIntMap(size), setupIntSlice(size)
}

func benchInt(size int) (elapsedSlice, elapsedMap time.Duration) {
	const tests = 5_000_000
	m, s := setupInt(size)
	// On average a lookup will take N/2 ops, let's make it
	// exact to avoid jitter.
	const avgFind = 2
	for range tests {
		now := time.Now()
		sliceHas(s, size/avgFind)
		elapsedSlice += time.Since(now)
	}
	for range tests {
		now := time.Now()
		mapHas(m, size/avgFind)
		elapsedMap += time.Since(now)
	}
	elapsedSlice /= tests
	elapsedMap /= tests
	fmt.Printf("\nsize=%v slice=%v map=%v ", size, elapsedSlice, elapsedMap)
	return elapsedSlice, elapsedMap
}

func cmpInt(size int) int {
	s, m := benchInt(size)
	if s+1*time.Nanosecond >= m && s-1*time.Nanosecond <= m {
		fmt.Printf("cmp=%v\n", 0)
		return 0
	}
	/*
		eps := 0.02
		if float64(s)*(1+eps) > float64(m) && float64(m)*(1+eps) > float64(s) {
			return 0
		}
	*/
	cmp := int(m - s)
	fmt.Printf("cmp=%v\n", cmp)
	return cmp
}

func isSliceFasterInt(size int) bool {
	return cmpInt(size) > 0
}
