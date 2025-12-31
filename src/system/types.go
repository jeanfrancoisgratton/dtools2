// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/31 00:12
// Original filename: src/system/types.go

package system

var JSONoutputfile = ""

type CatalogResponse struct {
	Repositories []string `json:"repositories"`
	// Some implementations also include this (not guaranteed everywhere).
	Next string `json:"next,omitempty"`
}

type TagsListResponse struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"` // may decode as nil/empty if no tags
}
