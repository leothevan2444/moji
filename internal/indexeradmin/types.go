// Package indexeradmin owns rule catalog and configured-source administration.
// Public projections contain no credentials or runtime download locators.
package indexeradmin

import (
	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/indexer"
)

type Value struct {
	String  *string
	Boolean *bool
	Number  *float64
}
type Option struct{ Value, Label string }
type Setting struct {
	Name, Type, Label  string
	Secret, Configured bool
	Value, Default     *Value
	Options            []Option
}
type Definition struct {
	LoginSupported                                   bool
	ID, Name, Description, Language, Privacy, Digest string
	Links, Modes, Warnings, Categories               []string
	Settings                                         []Setting
}
type DefinitionFilter struct {
	Query, Language, Privacy, Capability string
	Offset, Limit                        int
}
type DefinitionPage struct {
	Revision             string
	Total, Offset, Limit int
	Items                []Definition
}
type Instance struct {
	LoginSupported                                                         bool
	SiteURL, Privacy, Language, Description                                string
	ID, Name, DefinitionID, BaseURL, NetworkProfile, RuleDigest, ErrorCode string
	Kind                                                                   indexer.Kind
	Enabled                                                                bool
	TrustedOrigins                                                         []string
	Modes                                                                  []string
	Settings                                                               []Setting
	Health                                                                 indexer.HealthState
}
type InstancePage struct {
	Revision, RuleRevision, RuleErrorCode, CatalogErrorCode string
	JackettEnabled                                          bool
	JackettConfigured                                       bool
	JackettCatalog                                          indexer.CatalogStatus
	Order, NetworkProfiles                                  []string
	Items                                                   []Instance
	HealthErrorCode                                         string
}
type SaveResult struct {
	Revision string
	Instance Instance
}

type BuiltinPatch = config.BuiltinInstancePatch
