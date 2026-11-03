package indexerlogin

import "github.com/leothevan2444/moji/internal/indexer"

// AdapterFactory shares saved-site sessions with normal sourcing.
type AdapterFactory struct{ Sessions *indexer.Sessions }

func (f AdapterFactory) Open(p Prepared) (Execution, error) {
	if p.Instance.Kind != indexer.Builtin {
		return nil, indexer.ErrUnsupported
	}
	return indexer.NewLoginSession(f.Sessions, p.Instance, p.Plan)
}
