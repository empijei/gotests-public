package queues

import (
	"math/rand"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var impls = []struct {
	name string
	ctor func() Queue[int]
}{
	{"simple slice",
		func() Queue[int] {
			return &sliceQueue[int]{}
		},
	},
	{"ring slice",
		func() Queue[int] {
			return &ringQueue[int]{}
		},
	},
	{"chan backed",
		func() Queue[int] {
			return newChanQueue[int]()
		},
	},
	{"linked list",
		func() Queue[int] {
			var ll linkedListQueue[int]
			return &ll
		},
	},
	{"pooled linked list",
		func() Queue[int] {
			return newPooled[int]()
		},
	},
	{"map queue",
		func() Queue[int] {
			return newMapQueue[int]()
		},
	},
}

func TestQueues(t *testing.T) {
	enq := func(q Queue[int], qt int) {
		for i := range qt {
			q.Enqueue(i)
		}
	}
	deq := func(q Queue[int], qt int) {
		for range qt {
			q.Dequeue()
		}
	}
	tests := []struct {
		name string
		ops  func(Queue[int])
		want []int
	}{
		{
			name: "insert 5",
			ops: func(q Queue[int]) {
				enq(q, 5)
			},
			want: []int{0, 1, 2, 3, 4},
		},
		{
			name: "insert 5, pop 3, insert 3",
			ops: func(q Queue[int]) {
				enq(q, 5)
				deq(q, 3)
				enq(q, 3)
			},
			want: []int{3, 4, 0, 1, 2},
		},
		{
			name: "insert 5, pop 5, insert 3",
			ops: func(q Queue[int]) {
				enq(q, 5)
				deq(q, 5)
				enq(q, 3)
			},
			want: []int{0, 1, 2},
		},
		{
			name: "insert 5, pop 3, insert 5, pop 2",
			ops: func(q Queue[int]) {
				enq(q, 5)
				deq(q, 3)
				enq(q, 3)
				deq(q, 2)
			},
			want: []int{0, 1, 2},
		},
	}

	bakMin := minShrink
	bakBase := baseLen
	defer func() {
		minShrink = bakMin
		baseLen = bakBase
	}()
	minShrink = 2
	baseLen = 2

	for _, i := range impls {
		for _, tt := range tests {
			t.Run(i.name+"/"+tt.name, func(t *testing.T) {
				q := i.ctor()
				tt.ops(q)
				var got []int
				for q.Len() > 0 {
					got = append(got, q.Dequeue())
				}
				if diff := cmp.Diff(tt.want, got); diff != "" {
					t.Errorf("got %v want %v diff:\n%s", got, tt.want, diff)
				}
			})
		}
	}
}

const jitter = 10

var benchs = []struct {
	name string
	r    func(b *testing.B, qctor func() Queue[int], size int)
}{
	{"one by one empty", func(b *testing.B, qctor func() Queue[int], size int) {
		for range b.N {
			q := qctor()
			for range size {
				q.Enqueue(1)
				_ = q.Dequeue()
			}
			for q.Len() > 0 {
				_ = q.Dequeue()
			}
		}
	}},
	{"1 by 1 not empty", func(b *testing.B, qctor func() Queue[int], size int) {
		for range b.N {
			q := qctor()
			q.Enqueue(1)
			for range size {
				q.Enqueue(1)
				_ = q.Dequeue()
			}
			for q.Len() > 0 {
				_ = q.Dequeue()
			}
		}
	}},
	{"send first", func(b *testing.B, qctor func() Queue[int], size int) {
		for range b.N {
			q := qctor()
			q.Enqueue(1)
			for range size {
				q.Enqueue(1)
			}
			for q.Len() > 0 {
				_ = q.Dequeue()
			}
		}
	}},
	{"with jitter", func(b *testing.B, qctor func() Queue[int], size int) {
		for range b.N {
			q := qctor()
			for range jitter {
				for range rand.Intn(size / jitter) {
					q.Enqueue(1)
				}
				for range rand.Intn(size / jitter) {
					if q.Len() > 0 {
						_ = q.Dequeue()
					}
				}
			}
			for q.Len() > 0 {
				_ = q.Dequeue()
			}
		}
	}},
	{"more enq", func(b *testing.B, qctor func() Queue[int], size int) {
		for range b.N {
			q := qctor()
			for range jitter {
				for range rand.Intn(size/jitter) * 2 {
					q.Enqueue(1)
				}
				for range rand.Intn(size / jitter) {
					if q.Len() > 0 {
						_ = q.Dequeue()
					}
				}
			}
			for q.Len() > 0 {
				_ = q.Dequeue()
			}
		}
	}},
	{"more deq", func(b *testing.B, qctor func() Queue[int], size int) {
		for range b.N {
			q := qctor()
			for range jitter {
				for range rand.Intn(size / jitter) {
					q.Enqueue(1)
				}
				for range rand.Intn(size/jitter) * 2 {
					if q.Len() > 0 {
						_ = q.Dequeue()
					}
				}
			}
			for q.Len() > 0 {
				_ = q.Dequeue()
			}
		}
	}},
	{"grow and shrink", func(b *testing.B, qctor func() Queue[int], size int) {
		for range b.N {
			q := qctor()
			for range jitter {
				for range rand.Intn(size/jitter) * 2 {
					q.Enqueue(1)
				}
				for range rand.Intn(size / jitter) {
					if q.Len() > 0 {
						_ = q.Dequeue()
					}
				}
			}
			for range jitter {
				for range rand.Intn(size / jitter) {
					q.Enqueue(1)
				}
				for range rand.Intn(size/jitter) * 2 {
					if q.Len() > 0 {
						_ = q.Dequeue()
					}
				}
			}
			for q.Len() > 0 {
				_ = q.Dequeue()
			}
		}
	}},
}

/*
BenchmarkQueue/one_by_one_empty/10000000/simple_slice-14         	       9	 121843046 ns/op	80000496 B/op	10000003 allocs/op
BenchmarkQueue/one_by_one_empty/10000000/ring_slice-14           	      14	  83549027 ns/op	     112 B/op	       2 allocs/op
BenchmarkQueue/one_by_one_empty/10000000/chan_backed-14          	       5	 215711758 ns/op	     184 B/op	       2 allocs/op
BenchmarkQueue/one_by_one_empty/10000000/linked_list-14          	       7	 145332577 ns/op	160000275 B/op	10000003 allocs/op
BenchmarkQueue/one_by_one_empty/10000000/pooled_linked_list-14   	      13	  83949776 ns/op	    2179 B/op	       5 allocs/op
BenchmarkQueue/one_by_one_empty/10000000/map_queue-14            	       7	 144742196 ns/op	     216 B/op	       3 allocs/op

BenchmarkQueue/1_by_1_not_empty/10000000/simple_slice-14         	       7	 144838292 ns/op	160000273 B/op	10000004 allocs/op
BenchmarkQueue/1_by_1_not_empty/10000000/ring_slice-14           	      15	  83095636 ns/op	     112 B/op	       2 allocs/op
BenchmarkQueue/1_by_1_not_empty/10000000/chan_backed-14          	       5	 215882133 ns/op	     184 B/op	       2 allocs/op
BenchmarkQueue/1_by_1_not_empty/10000000/linked_list-14          	       7	 148270196 ns/op	160000355 B/op	10000005 allocs/op
BenchmarkQueue/1_by_1_not_empty/10000000/pooled_linked_list-14   	      13	  84722304 ns/op	    2371 B/op	       8 allocs/op
BenchmarkQueue/1_by_1_not_empty/10000000/map_queue-14            	       8	 126811703 ns/op	     216 B/op	       3 allocs/op

BenchmarkQueue/send_first/10000000/simple_slice-14               	      21	  53681673 ns/op	510582062 B/op	      59 allocs/op
BenchmarkQueue/send_first/10000000/ring_slice-14                 	      12	 104720910 ns/op	402651144 B/op	      39 allocs/op
BenchmarkQueue/send_first/10000000/chan_backed-14                	       2	 639467208 ns/op	402865192 B/op	      39 allocs/op
BenchmarkQueue/send_first/10000000/linked_list-14                	       5	 230276833 ns/op	160000078 B/op	10000002 allocs/op
BenchmarkQueue/send_first/10000000/pooled_linked_list-14         	       3	 451169153 ns/op	428442989 B/op	10000048 allocs/op
BenchmarkQueue/send_first/10000000/map_queue-14                  	       1	1717362625 ns/op	708244304 B/op	  308588 allocs/op

BenchmarkQueue/with_jitter/10000000/simple_slice-14              	      43	  25976665 ns/op	203996736 B/op	     149 allocs/op
BenchmarkQueue/with_jitter/10000000/ring_slice-14                	      34	  50340311 ns/op	112497365 B/op	      86 allocs/op
BenchmarkQueue/with_jitter/10000000/chan_backed-14               	       6	 230679451 ns/op	118721378 B/op	      82 allocs/op
BenchmarkQueue/with_jitter/10000000/linked_list-14               	      13	 107430776 ns/op	78420076 B/op	 4901250 allocs/op
BenchmarkQueue/with_jitter/10000000/pooled_linked_list-14        	       9	 138445241 ns/op	108763925 B/op	 1686032 allocs/op
BenchmarkQueue/with_jitter/10000000/map_queue-14                 	       3	 648132319 ns/op	154535824 B/op	  105371 allocs/op

BenchmarkQueue/more_enq/10000000/simple_slice-14                 	      27	  48590451 ns/op	424552035 B/op	      86 allocs/op
BenchmarkQueue/more_enq/10000000/ring_slice-14                   	      15	  92676703 ns/op	210496993 B/op	      51 allocs/op
BenchmarkQueue/more_enq/10000000/chan_backed-14                  	       3	 445120014 ns/op	210974333 B/op	      45 allocs/op
BenchmarkQueue/more_enq/10000000/linked_list-14                  	       5	 228798475 ns/op	168978436 B/op	10561151 allocs/op
BenchmarkQueue/more_enq/10000000/pooled_linked_list-14           	       4	 411148844 ns/op	321528792 B/op	 7248407 allocs/op
BenchmarkQueue/more_enq/10000000/map_queue-14                    	       1	2137234875 ns/op	723400560 B/op	  386194 allocs/op

BenchmarkQueue/more_deq/10000000/simple_slice-14                 	      38	  31169730 ns/op	205702099 B/op	     248 allocs/op
BenchmarkQueue/more_deq/10000000/ring_slice-14                   	      20	  55643169 ns/op	157599056 B/op	     187 allocs/op
BenchmarkQueue/more_deq/10000000/chan_backed-14                  	       3	 350532847 ns/op	202454749 B/op	     184 allocs/op
BenchmarkQueue/more_deq/10000000/linked_list-14                  	      12	 115935969 ns/op	79690968 B/op	 4980679 allocs/op
BenchmarkQueue/more_deq/10000000/pooled_linked_list-14           	      10	 127384396 ns/op	80124879 B/op	 1067165 allocs/op
BenchmarkQueue/more_deq/10000000/map_queue-14                    	       3	 403530625 ns/op	95681226 B/op	   78223 allocs/op

BenchmarkQueue/grow_and_shrink/10000000/simple_slice-14          	      16	  72690677 ns/op	599132662 B/op	     120 allocs/op
BenchmarkQueue/grow_and_shrink/10000000/ring_slice-14            	       9	 128928079 ns/op	244334117 B/op	      81 allocs/op
BenchmarkQueue/grow_and_shrink/10000000/chan_backed-14           	       2	 594684834 ns/op	215712728 B/op	      48 allocs/op
BenchmarkQueue/grow_and_shrink/10000000/linked_list-14           	       4	 362754198 ns/op	264604552 B/op	16537784 allocs/op
BenchmarkQueue/grow_and_shrink/10000000/pooled_linked_list-14    	       3	 411710236 ns/op	293733584 B/op	 6102257 allocs/op
BenchmarkQueue/grow_and_shrink/10000000/map_queue-14             	       1	1576088083 ns/op	358238912 B/op	  175534 allocs/op
*/
func BenchmarkQueue(b *testing.B) {
	b.ReportAllocs()
	// sizes := []int{100}
	sizes := []int{10_000_000}

	for _, t := range benchs {
		b.Run(t.name, func(b *testing.B) {
			for _, s := range sizes {
				b.Run(strconv.Itoa(s), func(b *testing.B) {
					for _, i := range impls {
						b.Run(i.name, func(b *testing.B) {
							t.r(b, i.ctor, s)
						})
					}
				})
			}
		})
	}
}
