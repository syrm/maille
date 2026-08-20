package web

import (
	"net/url"
	"testing"
)

func TestParseTransactionFilter(t *testing.T) {
	filter, page := parseTransactionFilter(url.Values{
		"q":        {"  amazon  "},
		"category": {"7"},
		"review":   {"uncategorized"},
		"page":     {"3"},
	})

	if filter.Search != "amazon" || filter.CategoryID != 7 || !filter.Uncategorized {
		t.Fatalf("filter = %#v", filter)
	}
	if page != 3 || filter.Limit != 25 || filter.Offset != 50 {
		t.Fatalf("page = %d, limit = %d, offset = %d", page, filter.Limit, filter.Offset)
	}
}
