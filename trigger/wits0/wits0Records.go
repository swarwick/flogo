package wits0

// wits0Records the WITS0 packet structure
type wits0Records struct {
	Records []wits0Record
}

// wits0Record the WITS0 record structure
type wits0Record struct {
	Record string
	Item   string
	Data   string
}
