package foo

type flagSet struct{}

func (flagSet) Bool(name string, value bool, usage string) *bool                     { return nil }
func (flagSet) BoolVar(p *bool, name string, value bool, usage string)               {}
func (flagSet) BoolP(name, short string, value bool, usage string) *bool             { return nil }
func (flagSet) StringVarP(p *string, name, short string, value string, usage string) {}
func (flagSet) StringVar(p *string, name string, value string, usage string)         {}

func Register(fs flagSet) {
	var b bool
	var s string
	fs.Bool("json", false, "")
	fs.BoolVar(&b, "json", false, "")
	fs.BoolP("force", "f", false, "")
	fs.StringVarP(&s, "force", "F", "", "")
	fs.BoolVar(&b, "verbose", false, "json")
	fs.Bool("jsonl", false, "")
	fs.StringVar(&s, "format", "json", "")
}
