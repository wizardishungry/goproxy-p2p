package localserver

import (
	"testing"

	"github.com/goproxy/goproxy"
)

func TestReshuffle(t *testing.T) {
	testCases := []struct {
		desc string
		idx  int
		size int
	}{
		{
			desc: "nothing happens",
			idx:  1,
			size: 2,
		},
		{
			desc: "index out of range",
			idx:  2,
			size: 2,
		},
		{
			desc: "last elem",
			idx:  2,
			size: 3,
		},
		{
			desc: "next to last elem",
			idx:  2,
			size: 4,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			mc := newMulticacher()
			list := []goproxy.Cacher{}

			for i := 0; i < tC.size; i++ {
				list = append(list, &cacherLogger{})
			}
			mc.Add(list...)
			mc.reshuffle(tC.idx)

			if len(mc.cachers) != tC.size {
				t.Errorf("%d, len(mc.cachers) != %d tC.size ", len(mc.cachers), tC.size)
			}
			if tC.idx < len(mc.cachers) && list[tC.idx] != mc.cachers[1] {
				t.Errorf("list[tC.idx] != mc.cachers[tC.idx]")
			}

			for i, needle := range list {
				found := 0
				for _, elem := range mc.cachers {
					if elem == needle {
						found++
					}
				}
				if found != 1 {
					t.Errorf("input cacher %d found %d times", i, found)
				}
			}
		})
	}
}
