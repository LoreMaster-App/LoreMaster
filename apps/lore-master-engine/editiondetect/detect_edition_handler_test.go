package editiondetect

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
)

func call(t *testing.T, baseURL string) (any, error) {
	t.Helper()
	params, err := json.Marshal(rpcprotocol.EditionDetectParams{BaseURL: baseURL})
	if err != nil {
		t.Fatal(err)
	}

	return DetectEdition(nil)(context.Background(), rpcserver.Call{Params: params})
}

func TestDetectEditionHandler(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/confluence/rest/applinks/1.0/manifest" {
			_, _ = w.Write([]byte("<manifest><typeId>confluence</typeId><version>8.5.3</version></manifest>"))

			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	result, err := call(t, server.URL+"/confluence")
	if err != nil {
		t.Fatal(err)
	}
	detected, ok := result.(rpcprotocol.EditionDetectResult)
	if !ok {
		t.Fatalf("result type %T", result)
	}
	if detected.Edition != "datacenter" || detected.Version != "8.5.3" {
		t.Fatalf("got %+v", detected)
	}
}

func TestDetectEditionHandlerUnreachable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	_, err := call(t, server.URL)
	var rpcError *rpcprotocol.Error
	if !errors.As(err, &rpcError) || rpcError.Code != rpcprotocol.CodePlatformUnreachable {
		t.Fatalf("err %v", err)
	}
}
