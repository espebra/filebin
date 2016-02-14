package model

// Sort files by DateTime
type FilesByDateTime []File

func (a FilesByDateTime) Len() int {
	return len(a)
}

func (a FilesByDateTime) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a FilesByDateTime) Less(i, j int) bool {
	return a[i].DateTime.Before(a[j].DateTime)
}

// Sort bins by Last Update
type BinsByLastUpdate []Bin

func (a BinsByLastUpdate) Len() int {
	return len(a)
}

func (a BinsByLastUpdate) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a BinsByLastUpdate) Less(i, j int) bool {
	return a[i].LastUpdateAt.After(a[j].LastUpdateAt)
}
