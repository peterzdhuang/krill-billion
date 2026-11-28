package data

// stringIntern deduplicates strings in the heap (labels, aliases, norms share
// one copy of each distinct value across the whole bank).
type stringIntern struct {
	m map[string]string
}

func newStringIntern(hint int) *stringIntern {
	if hint < 1024 {
		hint = 1024
	}
	return &stringIntern{m: make(map[string]string, hint)}
}

func (p *stringIntern) Get(s string) string {
	if s == "" {
		return ""
	}
	if v, ok := p.m[s]; ok {
		return v
	}
	p.m[s] = s
	return s
}
