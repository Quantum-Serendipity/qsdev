package cmdutil

type flagSet struct{}

func (flagSet) BoolVar(p *bool, name string, value bool, usage string) {}

func OutputFlag(fs flagSet, p *bool) { fs.BoolVar(p, "json", false, "JSON output") }
