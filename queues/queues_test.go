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
one_by_one_empty/10000000/ring_slice-14           	      16	  87710995 ns/op	     112 B/op	       2 allocs/op
one_by_one_empty/10000000/simple_slice-14         	       9	 128620500 ns/op	80000560 B/op	10000003 allocs/op
one_by_one_empty/10000000/chan_backed-14          	       5	 217515017 ns/op	     184 B/op	       2 allocs/op
one_by_one_empty/10000000/linked_list-14          	       7	 149063065 ns/op	160000257 B/op	10000003 allocs/op
one_by_one_empty/10000000/pooled_linked_list-14   	      13	 101179397 ns/op	    2179 B/op	       5 allocs/op

1_by_1_not_empty/10000000/ring_slice-14           	      14	  80320262 ns/op	     112 B/op	       2 allocs/op
1_by_1_not_empty/10000000/simple_slice-14         	       7	 144646185 ns/op	160000300 B/op	10000004 allocs/op
1_by_1_not_empty/10000000/chan_backed-14          	       5	 216541608 ns/op	     184 B/op	       2 allocs/op
1_by_1_not_empty/10000000/linked_list-14          	       8	 140864760 ns/op	160000280 B/op	10000004 allocs/op
1_by_1_not_empty/10000000/pooled_linked_list-14   	      14	  81481193 ns/op	    2369 B/op	       8 allocs/op

send_first/10000000/ring_slice-14                 	      10	 124528192 ns/op	402651120 B/op	      39 allocs/op
send_first/10000000/simple_slice-14               	      20	  57386581 ns/op	510582073 B/op	      59 allocs/op
send_first/10000000/chan_backed-14                	       2	 641076396 ns/op	402865288 B/op	      40 allocs/op
send_first/10000000/linked_list-14                	       5	 235952492 ns/op	160000078 B/op	10000002 allocs/op
send_first/10000000/pooled_linked_list-14         	       3	 447592500 ns/op	428442989 B/op	10000048 allocs/op

with_jitter/10000000/ring_slice-14                	      21	  52212335 ns/op	121354358 B/op	      85 allocs/op
with_jitter/10000000/simple_slice-14              	      69	  26384487 ns/op	201260108 B/op	     145 allocs/op
with_jitter/10000000/chan_backed-14               	       5	 252872092 ns/op	134837928 B/op	     101 allocs/op
with_jitter/10000000/linked_list-14               	      10	 116423667 ns/op	82208454 B/op	 5138026 allocs/op
with_jitter/10000000/pooled_linked_list-14        	       9	 121211861 ns/op	86891720 B/op	 1396691 allocs/op

more_enq/10000000/ring_slice-14                   	      12	  86706608 ns/op	200771429 B/op	      43 allocs/op
more_enq/10000000/simple_slice-14                 	      21	  55125442 ns/op	457113966 B/op	      70 allocs/op
more_enq/10000000/chan_backed-14                  	       3	 461585181 ns/op	209913682 B/op	      39 allocs/op
more_enq/10000000/linked_list-14                  	       6	 223444125 ns/op	153908712 B/op	 9619291 allocs/op
more_enq/10000000/pooled_linked_list-14           	       2	 515080188 ns/op	449214640 B/op	 8544519 allocs/op

more_deq/10000000/ring_slice-14                   	      21	  56245853 ns/op	143451255 B/op	     171 allocs/op
more_deq/10000000/simple_slice-14                 	      31	  32885551 ns/op	231541396 B/op	     229 allocs/op
more_deq/10000000/chan_backed-14                  	       4	 279621750 ns/op	148791408 B/op	     125 allocs/op
more_deq/10000000/linked_list-14                  	      10	 110187183 ns/op	78491835 B/op	 4905736 allocs/op
more_deq/10000000/pooled_linked_list-14           	       9	 116376778 ns/op	80446332 B/op	 1148550 allocs/op

grow_and_shrink/10000000/ring_slice-14            	       8	 150578932 ns/op	267714824 B/op	      90 allocs/op
grow_and_shrink/10000000/simple_slice-14          	      16	  83258049 ns/op	631710800 B/op	     121 allocs/op
grow_and_shrink/10000000/chan_backed-14           	       2	 566527979 ns/op	201522264 B/op	      37 allocs/op
grow_and_shrink/10000000/linked_list-14           	       4	 341054354 ns/op	244655424 B/op	15290958 allocs/op
grow_and_shrink/10000000/pooled_linked_list-14    	       3	 369013722 ns/op	269086562 B/op	 5632635 allocs/op
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
