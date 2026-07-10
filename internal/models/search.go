package models

type SearchResultGroup struct {
	Source SourceType
	Name   string
	Items  []Song
	Index  int
}

type SearchResults struct {
	Query  string
	Groups []SearchResultGroup
	Total  int
}
