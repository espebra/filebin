package model

// Sort files by DateTime
type ByDateTime []File

func (a ByDateTime) Len() int {
	return len(a)
}

func (a ByDateTime) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByDateTime) Less(i, j int) bool {
	return a[i].DateTime.Before(a[j].DateTime)
}
