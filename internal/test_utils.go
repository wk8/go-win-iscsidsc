package internal

// iterateOverAllSubsets will call f with all the 2^n - 1 (unordered) subsets of {0,1,2,...,n}
func IterateOverAllSubsets(n uint, f func(subset []uint)) {
	max := uint(1<<n - 1)
	subset := make([]uint, n)

	generateSubset := func(i uint) []uint {
		index := 0
		for j := uint(0); j < n; j++ {
			if i&(1<<j) != 0 {
				subset[index] = j
				index++
			}
		}
		return subset[:index]
	}

	for i := uint(1); i <= max; i++ {
		f(generateSubset(i))
	}
}
