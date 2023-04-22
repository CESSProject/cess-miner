package node

const MaxReplaceFiles = 30

const (
	Active = iota
	Calculate
	Missing
	Recovery
)

const (
	Cach_prefix_metadata = "metadata:"
	Cach_prefix_report   = "report:"
	Cach_prefix_idle     = "idle:"
)
