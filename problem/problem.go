package problem

import svn "github.com/assembla/svn2go"

// Problem represents a problem.
type Problem struct {
	Name string
}

// Repo returns the repository of the problem.
func (p *Problem) Repo() (*svn.Repo, error) {
	return svn.Open(p.Name)
}
