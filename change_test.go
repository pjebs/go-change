package change

import "testing"

func TestDetectChange(t *testing.T) {

	var tests = []struct {
		w   []float64
		idx int
	}{
		{
			[]float64{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
			0, // no change point found
		},

		{
			[]float64{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2},
			10, // the 1 before the first 2, due the scale factor
		},
		{
			[]float64{1, 1, 2, 2, 1, 1, 2, 2, 1, 1, 2, 3, 0, 1, 2, 2, 1, 1, 2, 2, 1, 1, 2},
			0, // change occurs but not statistically significant
		},
	}

	for _, tt := range tests {
		r := DetectChange(tt.w, 5, Conf95)
		if r.Difference == 0 && tt.idx == 0 {
			// no difference found and no difference expected -- good
		} else if r.Difference != 0 && r.Index == tt.idx {
			// difference found at expected location -- good
		} else {
			t.Errorf("DetectChange index=%d, wanted %d", r.Index, tt.idx)
		}
	}
}