package http_test

import (
	"bytes"
	"image"
	"image/png"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	filestorage "github.com/larb26656/my-feed/backend/internal/modules/file/service"
	sourcehttp "github.com/larb26656/my-feed/backend/internal/modules/source/adapter/http"
	sourcesqlite "github.com/larb26656/my-feed/backend/internal/modules/source/adapter/sqlite"
	sourcesvc "github.com/larb26656/my-feed/backend/internal/modules/source/service"
	rss "github.com/larb26656/my-feed/backend/internal/modules/sourcetype/adapter/rss"
	typesvc "github.com/larb26656/my-feed/backend/internal/modules/sourcetype/service"
	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
	"github.com/larb26656/my-feed/backend/internal/testkit"
	"github.com/larb26656/my-feed/backend/internal/testkit/api"
	"github.com/stretchr/testify/require"
)

func TestSourceProfileImageCanBeUploadedAndRetrieved(t *testing.T) {
	// Arrange
	router := api.New(t)
	src := createRSSSource(t, router, "News", "https://example.test/image")
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 1, 1))))
	data := buf.Bytes()
	path := "/api/v1/sources/" + src.ID + "/profile-image"
	// Act
	upload := httptest.NewRecorder()
	router.ServeHTTP(upload, httptest.NewRequest("PUT", path, bytes.NewReader(data)))
	// Assert
	require.Equal(t, 200, upload.Code, upload.Body.String())
	image := httptest.NewRecorder()
	router.ServeHTTP(image, httptest.NewRequest("GET", path, nil))
	require.Equal(t, 200, image.Code)
	require.Equal(t, data, image.Body.Bytes())
}

func TestInvalidReplacementPreservesExistingImage(t *testing.T) {
	// Arrange
	router := api.New(t)
	src := createRSSSource(t, router, "News", "https://example.test/keep")
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 1, 1))))
	path := "/api/v1/sources/" + src.ID + "/profile-image"
	put := func(data []byte) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("PUT", path, bytes.NewReader(data)))
		return w
	}
	// Act & Assert
	require.Equal(t, 200, put(buf.Bytes()).Code)
	require.Equal(t, 400, put([]byte("not an image")).Code)
	require.Equal(t, 400, put(make([]byte, 2*1024*1024+1)).Code)
	got := httptest.NewRecorder()
	router.ServeHTTP(got, httptest.NewRequest("GET", path, nil))
	require.Equal(t, 200, got.Code)
	require.Equal(t, buf.Bytes(), got.Body.Bytes())
}

func TestProfileImageSupportsDeactivatedSourcesAndRejectsManualOrUnknown(t *testing.T) {
	// Arrange
	router := api.New(t)
	src := createRSSSource(t, router, "Paused", "https://example.test/paused-image")
	testkit.Request(t, router, "POST", "/api/v1/sources/"+src.ID+"/deactivate", nil, 200)
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 1, 1))))
	put := func(id string) int {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("PUT", "/api/v1/sources/"+id+"/profile-image", bytes.NewReader(buf.Bytes())))
		return w.Code
	}
	// Act & Assert
	require.Equal(t, 200, put(src.ID))
	all := decodeSuccess[[]source](t, testkit.Request(t, router, "GET", "/api/v1/sources/", nil, 200))
	require.Equal(t, "manual", all[0].Type)
	require.Equal(t, 409, put(all[0].ID))
	require.Equal(t, 404, put(uuid.NewString()))
}

func TestRemoveSourceProfileImage(t *testing.T) {
	// Arrange
	router := api.New(t)
	src := createRSSSource(t, router, "News", "https://example.test/remove")
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 1, 1))))
	path := "/api/v1/sources/" + src.ID + "/profile-image"
	upload := httptest.NewRecorder()
	router.ServeHTTP(upload, httptest.NewRequest("PUT", path, bytes.NewReader(buf.Bytes())))
	require.Equal(t, 200, upload.Code)
	// Act
	removed := httptest.NewRecorder()
	router.ServeHTTP(removed, httptest.NewRequest("DELETE", path, nil))
	// Assert
	require.Equal(t, 200, removed.Code, removed.Body.String())
	got := httptest.NewRecorder()
	router.ServeHTTP(got, httptest.NewRequest("GET", path, nil))
	require.Equal(t, 404, got.Code)
	require.NotContains(t, removed.Body.String(), `"profileImageId":"`)
}

func TestRemoveProfileImageSucceedsWhenFileCleanupFails(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	router := api.NewWithUploads(t, dir)
	src := createRSSSource(t, router, "News", "https://example.test/cleanup")
	path := "/api/v1/sources/" + src.ID + "/profile-image"
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 1, 1))))
	upload := httptest.NewRecorder()
	router.ServeHTTP(upload, httptest.NewRequest("PUT", path, bytes.NewReader(buf.Bytes())))
	require.Equal(t, 200, upload.Code)
	files, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, files, 1)
	file := filepath.Join(dir, files[0].Name())
	require.NoError(t, os.Remove(file))
	require.NoError(t, os.Mkdir(file, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(file, "child"), []byte("x"), 0o600))
	// Act
	removed := httptest.NewRecorder()
	router.ServeHTTP(removed, httptest.NewRequest("DELETE", path, nil))
	// Assert
	require.Equal(t, 200, removed.Code, removed.Body.String())
	got := httptest.NewRecorder()
	router.ServeHTTP(got, httptest.NewRequest("GET", path, nil))
	require.Equal(t, 404, got.Code)
}

func TestRemoveProfileImageSupportsDeactivatedAndRejectsManualOrUnknown(t *testing.T) {
	// Arrange
	router := api.New(t)
	src := createRSSSource(t, router, "Paused", "https://example.test/remove-paused")
	testkit.Request(t, router, "POST", "/api/v1/sources/"+src.ID+"/deactivate", nil, 200)
	remove := func(id string) int {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("DELETE", "/api/v1/sources/"+id+"/profile-image", nil))
		return w.Code
	}
	// Act & Assert
	require.Equal(t, 200, remove(src.ID))
	all := decodeSuccess[[]source](t, testkit.Request(t, router, "GET", "/api/v1/sources/", nil, 200))
	require.Equal(t, "manual", all[0].Type)
	require.Equal(t, 409, remove(all[0].ID))
	require.Equal(t, 404, remove(uuid.NewString()))
}

func TestReplacingProfileImageServesNewImage(t *testing.T) {
	// Arrange
	router := api.New(t)
	src := createRSSSource(t, router, "News", "https://example.test/replace")
	path := "/api/v1/sources/" + src.ID + "/profile-image"
	var first, second bytes.Buffer
	require.NoError(t, png.Encode(&first, image.NewRGBA(image.Rect(0, 0, 1, 1))))
	require.NoError(t, png.Encode(&second, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	initial := httptest.NewRecorder()
	router.ServeHTTP(initial, httptest.NewRequest("PUT", path, bytes.NewReader(first.Bytes())))
	require.Equal(t, 200, initial.Code)

	// Act
	replacement := httptest.NewRecorder()
	router.ServeHTTP(replacement, httptest.NewRequest("PUT", path, bytes.NewReader(second.Bytes())))
	got := httptest.NewRecorder()
	router.ServeHTTP(got, httptest.NewRequest("GET", path, nil))

	// Assert
	require.Equal(t, 200, replacement.Code, replacement.Body.String())
	require.Equal(t, 200, got.Code)
	require.Equal(t, second.Bytes(), got.Body.Bytes())
}

func TestReplacementStorageFailurePreservesExistingImage(t *testing.T) {
	// Arrange
	dir := filepath.Join(t.TempDir(), "uploads")
	router := api.NewWithUploads(t, dir)
	src := createRSSSource(t, router, "News", "https://example.test/storage-failure")
	path := "/api/v1/sources/" + src.ID + "/profile-image"
	var data bytes.Buffer
	require.NoError(t, png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 1, 1))))
	initial := httptest.NewRecorder()
	router.ServeHTTP(initial, httptest.NewRequest("PUT", path, bytes.NewReader(data.Bytes())))
	require.Equal(t, 200, initial.Code)
	backup := dir + "-backup"
	require.NoError(t, os.Rename(dir, backup))
	require.NoError(t, os.WriteFile(dir, []byte("block directory"), 0o600))

	// Act
	failed := httptest.NewRecorder()
	router.ServeHTTP(failed, httptest.NewRequest("PUT", path, bytes.NewReader(data.Bytes())))
	require.NoError(t, os.Remove(dir))
	require.NoError(t, os.Rename(backup, dir))
	got := httptest.NewRecorder()
	router.ServeHTTP(got, httptest.NewRequest("GET", path, nil))

	// Assert
	require.Equal(t, 500, failed.Code)
	require.Contains(t, failed.Body.String(), `"code":"INTERNAL_SERVER_ERROR"`)
	require.NotContains(t, failed.Body.String(), dir)
	require.Equal(t, 200, got.Code)
	require.Equal(t, data.Bytes(), got.Body.Bytes())
}

func TestReplacementDatabaseFailurePreservesExistingImage(t *testing.T) {
	// Arrange
	pool := testkit.NewDatabase(t)
	svc := sourcesvc.New(sourcesqlite.NewRepository(pool.DB))
	registry, err := typesvc.New(rss.Definition(nil))
	require.NoError(t, err)
	svc.SetSourceTypes(registry)
	svc.SetFileStorage(filestorage.New(t.TempDir()))
	router := httpx.NewRouter()
	sourcehttp.NewHandler(svc, svc).RegisterRoutes(router.Group("/api/v1/sources"))
	src := createRSSSource(t, router, "News", "https://example.test/db-failure")
	path := "/api/v1/sources/" + src.ID + "/profile-image"
	var data bytes.Buffer
	require.NoError(t, png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 1, 1))))
	initial := httptest.NewRecorder()
	router.ServeHTTP(initial, httptest.NewRequest("PUT", path, bytes.NewReader(data.Bytes())))
	require.Equal(t, 200, initial.Code)
	_, err = pool.Exec(`CREATE TRIGGER fail_image_update BEFORE UPDATE OF profile_image_id ON sources BEGIN SELECT RAISE(FAIL, 'injected failure'); END`)
	require.NoError(t, err)
	// Act
	failed := httptest.NewRecorder()
	router.ServeHTTP(failed, httptest.NewRequest("PUT", path, bytes.NewReader(data.Bytes())))
	got := httptest.NewRecorder()
	router.ServeHTTP(got, httptest.NewRequest("GET", path, nil))
	// Assert
	require.Equal(t, 500, failed.Code)
	require.Contains(t, failed.Body.String(), `"code":"INTERNAL_SERVER_ERROR"`)
	require.NotContains(t, failed.Body.String(), "injected failure")
	require.Equal(t, 200, got.Code)
	require.Equal(t, data.Bytes(), got.Body.Bytes())
}
