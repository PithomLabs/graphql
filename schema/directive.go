package schema

import (
	"github.com/chirino/graphql/internal/lexer"
)

type Directive struct {
	Name    string
	NameLoc Location
	Args    ArgumentList
}

func (s *Directive) String() string {
	return FormatterToString(s)
}

func ParseDirectives(l *lexer.Lexer) DirectiveList {
	var directives DirectiveList
	for l.Peek() == '@' {
		l.ConsumeToken('@')
		d := &Directive{}
		d.Name, d.NameLoc = l.ConsumeIdentInternWithLoc()
		d.NameLoc.Column--
		if l.Peek() == '(' {
			d.Args = ParseArguments(l)
		}
		directives = append(directives, d)
	}
	return directives
}

type DirectiveList []*Directive

func (l DirectiveList) Get(name string) *Directive {
	for _, d := range l {
		if d.Name == name {
			return d
		}
	}
	return nil
}

func (l DirectiveList) Select(keep func(d *Directive) bool) DirectiveList {
	rc := DirectiveList{}
	for _, d := range l {
		if keep(d) {
			rc = append(rc, d)
		}
	}
	return rc
}
