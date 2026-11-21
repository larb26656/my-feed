package cursor_test

import (
	"encoding/json"
	"testing"

	"github.com/larb26656/my-feed/backend/internal/platform/cursor"
	"github.com/stretchr/testify/require"
)

func TestCodecPreservesCursorEncoding(t *testing.T) {
	type payload struct {
		ID string `json:"id"`
	}
	// Raw URL-safe Base64 of {"id":"??"}, containing "_" and no padding.
	const existingToken = "eyJpZCI6Ij8_In0"
	want := payload{ID: "??"}

	token, err := cursor.Encode(want)
	require.NoError(t, err)
	require.Equal(t, existingToken, token)

	var got payload
	require.NoError(t, cursor.Decode(existingToken, &got))
	require.Equal(t, want, got)
}

func TestDecodeRejectsInvalidTokens(t *testing.T) {
	for _, test := range []struct {
		name  string
		token string
	}{
		{name: "invalid base64", token: "%%"},
		{name: "invalid JSON", token: "ew"},          // {
		{name: "incompatible payload", token: "W10"}, // []
	} {
		t.Run(test.name, func(t *testing.T) {
			var payload struct{ ID string }
			require.Error(t, cursor.Decode(test.token, &payload))
		})
	}
}

func TestEncodeReturnsMarshalErrors(t *testing.T) {
	token, err := cursor.Encode(make(chan int))
	var unsupported *json.UnsupportedTypeError
	require.ErrorAs(t, err, &unsupported)
	require.Empty(t, token)
}
