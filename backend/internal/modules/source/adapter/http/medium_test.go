package http_test

import (
	"testing"

	"github.com/larb26656/my-feed/backend/internal/modules/source/domain"
	"github.com/larb26656/my-feed/backend/internal/modules/sourcetype/adapter/medium"
	"github.com/larb26656/my-feed/backend/internal/testkit"
	"github.com/larb26656/my-feed/backend/internal/testkit/api"
	"github.com/stretchr/testify/require"
)

func TestMediumCreateNormalizesAndConflictsWhenDeactivated(t *testing.T) {
	// Arrange
	router := api.NewWithDefinitions(t, t.TempDir(), medium.Definition(nil))

	// Act
	created := decodeSuccess[source](t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]any{
		"name": "Alice", "type": string(domain.TypeMedium), "config": map[string]any{"kind": "profile", "identifier": " @Alice "},
	}, 201))

	// Assert
	require.Equal(t, string(domain.TypeMedium), created.Type)
	require.Equal(t, map[string]any{"kind": "profile", "identifier": "Alice"}, created.Config)

	// Act: duplicate active target
	failure := decodeFailure(t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]any{
		"name": "Dup", "type": string(domain.TypeMedium), "config": map[string]any{"kind": "profile", "identifier": "Alice"},
	}, 409))

	// Assert
	require.Equal(t, "CONFLICT", failure.Code)
	require.Contains(t, failure.Message, "medium source already in use")

	// Arrange: deactivate then recreate same target
	testkit.Request(t, router, "POST", "/api/v1/sources/"+created.ID+"/deactivate", nil, 200)

	// Act
	again := decodeFailure(t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]any{
		"name": "Still dup", "type": string(domain.TypeMedium), "config": map[string]any{"kind": "profile", "identifier": "@Alice"},
	}, 409))

	// Assert
	require.Equal(t, "CONFLICT", again.Code)
}

func TestMediumPublicationCreateConflictsIncludingDeactivated(t *testing.T) {
	// Arrange
	router := api.NewWithDefinitions(t, t.TempDir(), medium.Definition(nil))

	// Act
	created := decodeSuccess[source](t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]any{
		"name": "Alice Pub", "type": string(domain.TypeMedium), "config": map[string]any{"kind": "publication", "identifier": " Alice "},
	}, 201))

	// Assert
	require.Equal(t, map[string]any{"kind": "publication", "identifier": "Alice"}, created.Config)

	// Act: reject leading @ for publication
	rejectAt := decodeFailure(t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]any{
		"name": "Bad", "type": string(domain.TypeMedium), "config": map[string]any{"kind": "publication", "identifier": "@alice"},
	}, 400))

	// Assert
	require.Equal(t, "VALIDATION", rejectAt.Code)

	// Act: duplicate active publication
	failure := decodeFailure(t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]any{
		"name": "Dup", "type": string(domain.TypeMedium), "config": map[string]any{"kind": "publication", "identifier": "Alice"},
	}, 409))

	// Assert
	require.Equal(t, "CONFLICT", failure.Code)

	// Arrange: deactivate then recreate same target
	testkit.Request(t, router, "POST", "/api/v1/sources/"+created.ID+"/deactivate", nil, 200)

	// Act
	again := decodeFailure(t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]any{
		"name": "Still dup", "type": string(domain.TypeMedium), "config": map[string]any{"kind": "publication", "identifier": "Alice"},
	}, 409))

	// Assert
	require.Equal(t, "CONFLICT", again.Code)
}

func TestMediumProfileAndPublicationSameIdentifierCoexist(t *testing.T) {
	// Arrange
	router := api.NewWithDefinitions(t, t.TempDir(), medium.Definition(nil))

	// Act
	profile := decodeSuccess[source](t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]any{
		"name": "Alice Profile", "type": string(domain.TypeMedium), "config": map[string]any{"kind": "profile", "identifier": "alice"},
	}, 201))
	publication := decodeSuccess[source](t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]any{
		"name": "Alice Pub", "type": string(domain.TypeMedium), "config": map[string]any{"kind": "publication", "identifier": "alice"},
	}, 201))

	// Assert
	require.Equal(t, map[string]any{"kind": "profile", "identifier": "alice"}, profile.Config)
	require.Equal(t, map[string]any{"kind": "publication", "identifier": "alice"}, publication.Config)
	require.NotEqual(t, profile.ID, publication.ID)
}

func TestMediumCoexistsWithRSSSameFeedURL(t *testing.T) {
	// Arrange
	router := api.NewWithDefinitions(t, t.TempDir(), medium.Definition(nil))

	// Act
	mediumSrc := decodeSuccess[source](t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]any{
		"name": "Alice Pub", "type": string(domain.TypeMedium), "config": map[string]any{"kind": "publication", "identifier": "alice"},
	}, 201))
	rssSrc := decodeSuccess[source](t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]any{
		"name": "Alice RSS", "type": "rss", "config": map[string]any{"url": "https://medium.com/feed/alice"},
	}, 201))

	// Assert
	require.Equal(t, string(domain.TypeMedium), mediumSrc.Type)
	require.Equal(t, "rss", rssSrc.Type)
}

func TestMediumRenameAllowedAndTargetChangeRejected(t *testing.T) {
	// Arrange
	router := api.NewWithDefinitions(t, t.TempDir(), medium.Definition(nil))
	created := decodeSuccess[source](t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]any{
		"name": "Alice", "type": string(domain.TypeMedium), "config": map[string]any{"kind": "profile", "identifier": "alice"},
	}, 201))

	// Act: rename with same normalized target
	renamed := decodeSuccess[source](t, testkit.Request(t, router, "PATCH", "/api/v1/sources/"+created.ID, map[string]any{
		"name": "Renamed", "config": map[string]any{"kind": "profile", "identifier": "@alice"},
	}, 200))

	// Assert
	require.Equal(t, "Renamed", renamed.Name)
	require.Equal(t, map[string]any{"kind": "profile", "identifier": "alice"}, renamed.Config)

	// Act: name-only update
	nameOnly := decodeSuccess[source](t, testkit.Request(t, router, "PATCH", "/api/v1/sources/"+created.ID, map[string]any{
		"name": "Name only",
	}, 200))

	// Assert
	require.Equal(t, "Name only", nameOnly.Name)
	require.Equal(t, map[string]any{"kind": "profile", "identifier": "alice"}, nameOnly.Config)

	// Act: reject identifier change
	failure := decodeFailure(t, testkit.Request(t, router, "PATCH", "/api/v1/sources/"+created.ID, map[string]any{
		"name": "Changed", "config": map[string]any{"kind": "profile", "identifier": "bob"},
	}, 400))

	// Assert
	require.Equal(t, "VALIDATION", failure.Code)
	require.Contains(t, failure.Message, "medium source target cannot be changed")

	// Act: reject kind change
	kindChange := decodeFailure(t, testkit.Request(t, router, "PATCH", "/api/v1/sources/"+created.ID, map[string]any{
		"name": "Kind change", "config": map[string]any{"kind": "publication", "identifier": "alice"},
	}, 400))

	// Assert
	require.Equal(t, "VALIDATION", kindChange.Code)
	require.Contains(t, kindChange.Message, "medium source target cannot be changed")

	got := decodeSuccess[source](t, testkit.Request(t, router, "GET", "/api/v1/sources/"+created.ID, nil, 200))
	require.Equal(t, "Name only", got.Name)
	require.Equal(t, map[string]any{"kind": "profile", "identifier": "alice"}, got.Config)
}

func TestMediumPublicationRenameAllowedAndTargetChangeRejected(t *testing.T) {
	// Arrange
	router := api.NewWithDefinitions(t, t.TempDir(), medium.Definition(nil))
	created := decodeSuccess[source](t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]any{
		"name": "Alice Pub", "type": string(domain.TypeMedium), "config": map[string]any{"kind": "publication", "identifier": "alice"},
	}, 201))

	// Act: rename with same target
	renamed := decodeSuccess[source](t, testkit.Request(t, router, "PATCH", "/api/v1/sources/"+created.ID, map[string]any{
		"name": "Renamed Pub", "config": map[string]any{"kind": "publication", "identifier": "alice"},
	}, 200))

	// Assert
	require.Equal(t, "Renamed Pub", renamed.Name)
	require.Equal(t, map[string]any{"kind": "publication", "identifier": "alice"}, renamed.Config)

	// Act: reject slug change
	slugChange := decodeFailure(t, testkit.Request(t, router, "PATCH", "/api/v1/sources/"+created.ID, map[string]any{
		"name": "Changed", "config": map[string]any{"kind": "publication", "identifier": "bob"},
	}, 400))

	// Assert
	require.Equal(t, "VALIDATION", slugChange.Code)
	require.Contains(t, slugChange.Message, "medium source target cannot be changed")

	// Act: reject kind change
	kindChange := decodeFailure(t, testkit.Request(t, router, "PATCH", "/api/v1/sources/"+created.ID, map[string]any{
		"name": "Kind change", "config": map[string]any{"kind": "profile", "identifier": "alice"},
	}, 400))

	// Assert
	require.Equal(t, "VALIDATION", kindChange.Code)

	got := decodeSuccess[source](t, testkit.Request(t, router, "GET", "/api/v1/sources/"+created.ID, nil, 200))
	require.Equal(t, "Renamed Pub", got.Name)
	require.Equal(t, map[string]any{"kind": "publication", "identifier": "alice"}, got.Config)
}

func TestMediumRejectedTargetChangePreservesExistingEntries(t *testing.T) {
	// Arrange
	router := api.NewWithDefinitions(t, t.TempDir(), medium.Definition(nil))
	created := decodeSuccess[source](t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]any{
		"name": "Alice", "type": string(domain.TypeMedium), "config": map[string]any{"kind": "profile", "identifier": "alice"},
	}, 201))
	entry := decodeSuccess[struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		Link  string `json:"link"`
	}](t, testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]any{
		"sourceId": created.ID,
		"title":    "Story",
		"link":     "https://medium.com/@alice/story",
	}, 201))

	// Act
	failure := decodeFailure(t, testkit.Request(t, router, "PATCH", "/api/v1/sources/"+created.ID, map[string]any{
		"name": "Changed", "config": map[string]any{"kind": "profile", "identifier": "bob"},
	}, 400))

	// Assert
	require.Equal(t, "VALIDATION", failure.Code)
	require.Contains(t, failure.Message, "medium source target cannot be changed")

	listed := decodeSuccess[[]struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		Link     string `json:"link"`
		SourceID string `json:"sourceId"`
	}](t, testkit.Request(t, router, "GET", "/api/v1/entries/?sourceId="+created.ID, nil, 200))
	require.Len(t, listed, 1)
	require.Equal(t, entry.ID, listed[0].ID)
	require.Equal(t, "Story", listed[0].Title)
	require.Equal(t, "https://medium.com/@alice/story", listed[0].Link)
	require.Equal(t, created.ID, listed[0].SourceID)

	got := decodeSuccess[source](t, testkit.Request(t, router, "GET", "/api/v1/sources/"+created.ID, nil, 200))
	require.Equal(t, map[string]any{"kind": "profile", "identifier": "alice"}, got.Config)
}
