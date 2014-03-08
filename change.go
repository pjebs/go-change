// Package change implements an online change detection algorithm
/*
http://excelsior.cs.ucsb.edu/papers/as06.pdf

How effective is this algorithm?  The authors state
"[T]he estimation algorithm can be shown theoretically to give the correct
prediction and approach an unbiased estimator of the true changeover point if
the window length approaches infinity and the two distributions are
sufficiently dissimilar."

The algorithm works by examining the distributions on either side of a
suspected change point, and finding the index in the window where the two
distributions are most dissimilar.  As there is not guaranteed to be a change
point in the window, this implementation also performs a Student's t-test on
the two distributions to reduce the rate of false positives.

*/
package change

import "math"

type Stats struct {
	Mean     float64
	Variance float64
	N        int
}

type ChangePoint struct {
	Index   int
	TResult TResult
	Before  Stats
	After   Stats
}

// TODO(dgryski): move some of the params to a struct so we just have detector.Check(window)

// DetectChange returns the index of a potential change point
func DetectChange(window []float64, minSampleSize int, tConf Confidence) *ChangePoint {

	n := len(window)

	// The paper provides recursive formulas for computing the means and
	// standard deviations as we slide along the window.  This
	// implementation uses alternate math based on cumulative sums.

	// cumsum contains the cumulative sum of all elements <= i
	// cumsumsq contains the cumulative sum of squares of all elements <= i
	cumsum := make([]float64, 0, n)
	cumsumsq := make([]float64, 0, n)

	var sum, sumsq float64
	for _, v := range window {
		sum += v
		sumsq += v * v
		cumsum = append(cumsum, sum)
		cumsumsq = append(cumsumsq, sumsq)
	}

	// sb is our between-class scatter, the degree of dissimilarity of the
	// two distributions.  This value is always positive, so we can set 0
	// as the minimum and know that any valid value will be larger
	var maxsb float64
	var maxsbIdx int

	// The paper also provides a metric sw, for 'within-class scatter',
	// which depends on the standard-deviation of the samples. The suggests
	// finding the point that minimizes the ratio sw/sb.  However, it then
	// proves that this is equivalent to maximizing sb.  The calculation of
	// sb depends only on the means of the two samples, and not of the
	// variances.  However, we calculate the variances so that we can pass
	// them to the T test later on.

	var before, after Stats

	for l := minSampleSize; l < (n - 1 - minSampleSize); l++ {
		n1 := float64(l + 1)
		mean1 := cumsum[l] / n1

		n2 := float64(n - l - 1)
		sum2 := (sum - cumsum[l])
		mean2 := sum2 / n2

		sb := ((n1 * n2) / (n1 + n2)) * (mean1 - mean2) * (mean1 - mean2)
		if maxsb < sb {
			maxsb = sb
			maxsbIdx = l

			// The variances are calculated only if needed to
			// reduce the main in the main loop
			var1 := (cumsumsq[l] - (cumsum[l]*cumsum[l])/(n1-1)) / (n1 - 1)
			var2 := ((sumsq - cumsumsq[l]) - (sum2*sum2)/(n2-1)) / (n2 - 1)

			before.Mean, before.Variance, before.N = mean1, var1, l+1
			after.Mean, after.Variance, after.N = mean2, var2, n-l-1
		}
	}

	cp := &ChangePoint{
		Index:   maxsbIdx,
		TResult: ttest(before, after, tConf),
		Before:  before,
		After:   after,
	}

	return cp
}

// TResult is the result of a Student's t-test.
type TResult struct {
	// The absolute difference in sample means
	Difference float64

	// The percentage difference in sample means
	Percent float64
}

// From https://github.com/codahale/ministat/blob/master/src/ministat.c
func ttest(ds, rs Stats, confidx Confidence) TResult {

	i := ds.N + rs.N - 2

	var t float64

	if i > nstudent {
		t = student[0][confidx]
	} else {
		t = student[i][confidx]
	}

	dn := float64(ds.N)
	rn := float64(rs.N)

	spool := (dn-1)*ds.Variance + (rn-1)*rs.Variance
	spool /= dn + rn - 2
	spool = math.Sqrt(spool)
	s := spool * math.Sqrt(1/dn+1/rn)
	d := ds.Mean - rs.Mean
	e := t * s

	if math.Abs(d) <= e {
		// no difference
		return TResult{}
	}

	return TResult{
		Difference: d,
		Percent:    d * 100 / rs.Mean,
	}
}

// Confidence is a confidence level for the Student's t-test
type Confidence int

const (
	Conf80 Confidence = iota
	Conf90
	Conf95
	Conf98
	Conf99
	Conf99p5
)

const nconf = 6

var studentpct = [nconf]float64{80, 90, 95, 98, 99, 99.5}

const nstudent = 100

// +1 because 0 is used for 'more than 100 samples'
var student = [nstudent + 1][nconf]float64{
	/* inf */ {1.282, 1.645, 1.960, 2.326, 2.576, 3.090},
	/* 1. */ {3.078, 6.314, 12.706, 31.821, 63.657, 318.313},
	/* 2. */ {1.886, 2.920, 4.303, 6.965, 9.925, 22.327},
	/* 3. */ {1.638, 2.353, 3.182, 4.541, 5.841, 10.215},
	/* 4. */ {1.533, 2.132, 2.776, 3.747, 4.604, 7.173},
	/* 5. */ {1.476, 2.015, 2.571, 3.365, 4.032, 5.893},
	/* 6. */ {1.440, 1.943, 2.447, 3.143, 3.707, 5.208},
	/* 7. */ {1.415, 1.895, 2.365, 2.998, 3.499, 4.782},
	/* 8. */ {1.397, 1.860, 2.306, 2.896, 3.355, 4.499},
	/* 9. */ {1.383, 1.833, 2.262, 2.821, 3.250, 4.296},
	/* 10. */ {1.372, 1.812, 2.228, 2.764, 3.169, 4.143},
	/* 11. */ {1.363, 1.796, 2.201, 2.718, 3.106, 4.024},
	/* 12. */ {1.356, 1.782, 2.179, 2.681, 3.055, 3.929},
	/* 13. */ {1.350, 1.771, 2.160, 2.650, 3.012, 3.852},
	/* 14. */ {1.345, 1.761, 2.145, 2.624, 2.977, 3.787},
	/* 15. */ {1.341, 1.753, 2.131, 2.602, 2.947, 3.733},
	/* 16. */ {1.337, 1.746, 2.120, 2.583, 2.921, 3.686},
	/* 17. */ {1.333, 1.740, 2.110, 2.567, 2.898, 3.646},
	/* 18. */ {1.330, 1.734, 2.101, 2.552, 2.878, 3.610},
	/* 19. */ {1.328, 1.729, 2.093, 2.539, 2.861, 3.579},
	/* 20. */ {1.325, 1.725, 2.086, 2.528, 2.845, 3.552},
	/* 21. */ {1.323, 1.721, 2.080, 2.518, 2.831, 3.527},
	/* 22. */ {1.321, 1.717, 2.074, 2.508, 2.819, 3.505},
	/* 23. */ {1.319, 1.714, 2.069, 2.500, 2.807, 3.485},
	/* 24. */ {1.318, 1.711, 2.064, 2.492, 2.797, 3.467},
	/* 25. */ {1.316, 1.708, 2.060, 2.485, 2.787, 3.450},
	/* 26. */ {1.315, 1.706, 2.056, 2.479, 2.779, 3.435},
	/* 27. */ {1.314, 1.703, 2.052, 2.473, 2.771, 3.421},
	/* 28. */ {1.313, 1.701, 2.048, 2.467, 2.763, 3.408},
	/* 29. */ {1.311, 1.699, 2.045, 2.462, 2.756, 3.396},
	/* 30. */ {1.310, 1.697, 2.042, 2.457, 2.750, 3.385},
	/* 31. */ {1.309, 1.696, 2.040, 2.453, 2.744, 3.375},
	/* 32. */ {1.309, 1.694, 2.037, 2.449, 2.738, 3.365},
	/* 33. */ {1.308, 1.692, 2.035, 2.445, 2.733, 3.356},
	/* 34. */ {1.307, 1.691, 2.032, 2.441, 2.728, 3.348},
	/* 35. */ {1.306, 1.690, 2.030, 2.438, 2.724, 3.340},
	/* 36. */ {1.306, 1.688, 2.028, 2.434, 2.719, 3.333},
	/* 37. */ {1.305, 1.687, 2.026, 2.431, 2.715, 3.326},
	/* 38. */ {1.304, 1.686, 2.024, 2.429, 2.712, 3.319},
	/* 39. */ {1.304, 1.685, 2.023, 2.426, 2.708, 3.313},
	/* 40. */ {1.303, 1.684, 2.021, 2.423, 2.704, 3.307},
	/* 41. */ {1.303, 1.683, 2.020, 2.421, 2.701, 3.301},
	/* 42. */ {1.302, 1.682, 2.018, 2.418, 2.698, 3.296},
	/* 43. */ {1.302, 1.681, 2.017, 2.416, 2.695, 3.291},
	/* 44. */ {1.301, 1.680, 2.015, 2.414, 2.692, 3.286},
	/* 45. */ {1.301, 1.679, 2.014, 2.412, 2.690, 3.281},
	/* 46. */ {1.300, 1.679, 2.013, 2.410, 2.687, 3.277},
	/* 47. */ {1.300, 1.678, 2.012, 2.408, 2.685, 3.273},
	/* 48. */ {1.299, 1.677, 2.011, 2.407, 2.682, 3.269},
	/* 49. */ {1.299, 1.677, 2.010, 2.405, 2.680, 3.265},
	/* 50. */ {1.299, 1.676, 2.009, 2.403, 2.678, 3.261},
	/* 51. */ {1.298, 1.675, 2.008, 2.402, 2.676, 3.258},
	/* 52. */ {1.298, 1.675, 2.007, 2.400, 2.674, 3.255},
	/* 53. */ {1.298, 1.674, 2.006, 2.399, 2.672, 3.251},
	/* 54. */ {1.297, 1.674, 2.005, 2.397, 2.670, 3.248},
	/* 55. */ {1.297, 1.673, 2.004, 2.396, 2.668, 3.245},
	/* 56. */ {1.297, 1.673, 2.003, 2.395, 2.667, 3.242},
	/* 57. */ {1.297, 1.672, 2.002, 2.394, 2.665, 3.239},
	/* 58. */ {1.296, 1.672, 2.002, 2.392, 2.663, 3.237},
	/* 59. */ {1.296, 1.671, 2.001, 2.391, 2.662, 3.234},
	/* 60. */ {1.296, 1.671, 2.000, 2.390, 2.660, 3.232},
	/* 61. */ {1.296, 1.670, 2.000, 2.389, 2.659, 3.229},
	/* 62. */ {1.295, 1.670, 1.999, 2.388, 2.657, 3.227},
	/* 63. */ {1.295, 1.669, 1.998, 2.387, 2.656, 3.225},
	/* 64. */ {1.295, 1.669, 1.998, 2.386, 2.655, 3.223},
	/* 65. */ {1.295, 1.669, 1.997, 2.385, 2.654, 3.220},
	/* 66. */ {1.295, 1.668, 1.997, 2.384, 2.652, 3.218},
	/* 67. */ {1.294, 1.668, 1.996, 2.383, 2.651, 3.216},
	/* 68. */ {1.294, 1.668, 1.995, 2.382, 2.650, 3.214},
	/* 69. */ {1.294, 1.667, 1.995, 2.382, 2.649, 3.213},
	/* 70. */ {1.294, 1.667, 1.994, 2.381, 2.648, 3.211},
	/* 71. */ {1.294, 1.667, 1.994, 2.380, 2.647, 3.209},
	/* 72. */ {1.293, 1.666, 1.993, 2.379, 2.646, 3.207},
	/* 73. */ {1.293, 1.666, 1.993, 2.379, 2.645, 3.206},
	/* 74. */ {1.293, 1.666, 1.993, 2.378, 2.644, 3.204},
	/* 75. */ {1.293, 1.665, 1.992, 2.377, 2.643, 3.202},
	/* 76. */ {1.293, 1.665, 1.992, 2.376, 2.642, 3.201},
	/* 77. */ {1.293, 1.665, 1.991, 2.376, 2.641, 3.199},
	/* 78. */ {1.292, 1.665, 1.991, 2.375, 2.640, 3.198},
	/* 79. */ {1.292, 1.664, 1.990, 2.374, 2.640, 3.197},
	/* 80. */ {1.292, 1.664, 1.990, 2.374, 2.639, 3.195},
	/* 81. */ {1.292, 1.664, 1.990, 2.373, 2.638, 3.194},
	/* 82. */ {1.292, 1.664, 1.989, 2.373, 2.637, 3.193},
	/* 83. */ {1.292, 1.663, 1.989, 2.372, 2.636, 3.191},
	/* 84. */ {1.292, 1.663, 1.989, 2.372, 2.636, 3.190},
	/* 85. */ {1.292, 1.663, 1.988, 2.371, 2.635, 3.189},
	/* 86. */ {1.291, 1.663, 1.988, 2.370, 2.634, 3.188},
	/* 87. */ {1.291, 1.663, 1.988, 2.370, 2.634, 3.187},
	/* 88. */ {1.291, 1.662, 1.987, 2.369, 2.633, 3.185},
	/* 89. */ {1.291, 1.662, 1.987, 2.369, 2.632, 3.184},
	/* 90. */ {1.291, 1.662, 1.987, 2.368, 2.632, 3.183},
	/* 91. */ {1.291, 1.662, 1.986, 2.368, 2.631, 3.182},
	/* 92. */ {1.291, 1.662, 1.986, 2.368, 2.630, 3.181},
	/* 93. */ {1.291, 1.661, 1.986, 2.367, 2.630, 3.180},
	/* 94. */ {1.291, 1.661, 1.986, 2.367, 2.629, 3.179},
	/* 95. */ {1.291, 1.661, 1.985, 2.366, 2.629, 3.178},
	/* 96. */ {1.290, 1.661, 1.985, 2.366, 2.628, 3.177},
	/* 97. */ {1.290, 1.661, 1.985, 2.365, 2.627, 3.176},
	/* 98. */ {1.290, 1.661, 1.984, 2.365, 2.627, 3.175},
	/* 99. */ {1.290, 1.660, 1.984, 2.365, 2.626, 3.175},
	/* 100. */ {1.290, 1.660, 1.984, 2.364, 2.626, 3.174},
}