package catalog

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVariantsGroupCompleteShardsAndProjector(t *testing.T) {
	sha := strings.Repeat("a", 40)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/tree/") {
			fmt.Fprintf(w, `{"sha":%q,"gated":false}`, sha)
			return
		}
		fmt.Fprint(w, `[
            {"type":"file","path":"model-Q4_K_M-00002-of-00002.gguf","size":30},
            {"type":"file","path":"model-Q4_K_M-00001-of-00002.gguf","size":20},
            {"type":"file","path":"model-Q8_0-00001-of-00002.gguf","size":40},
            {"type":"file","path":"mmproj-f16.gguf","size":10},
            {"type":"file","path":"draft.gguf","size":5},
            {"type":"file","path":"MTP/mtp-Qwen3.8-Flash-Next-Q4_K_M.gguf","size":5},
            {"type":"file","path":"mtp-model.gguf","size":5},
            {"type":"file","path":"DeepSeek-V4-Flash-Vision-Encoder.gguf","size":5},
            {"type":"file","path":"imatrix-qwen3.8-27b.gguf","size":5},
            {"type":"file","path":"../escape.gguf","size":5}
        ]`)
	}))
	defer server.Close()
	client := New()
	client.BaseURL = server.URL
	repo, err := client.Repository(context.Background(), "owner/model")
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.Variants) != 1 {
		t.Fatal(repo)
	}
	variant := repo.Variants[0]
	if variant.Size != 60 || len(variant.Files) != 3 || variant.Projector != "mmproj-f16.gguf" || !strings.Contains(variant.Files[0].URL, sha) {
		t.Fatal(variant)
	}
}

func TestRepositoryFollowsCachedTreePages(t *testing.T) {
	sha := strings.Repeat("b", 40)
	var address string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/tree/") {
			fmt.Fprintf(w, `{"sha":%q,"gated":false}`, sha)
			return
		}
		if r.URL.Query().Get("cursor") == "next" {
			fmt.Fprint(w, `[{"type":"file","path":"model-Q4_K_M-00002-of-00002.gguf","size":20}]`)
			return
		}
		w.Header().Set("Link", "<"+address+r.URL.Path+`?cursor=next>; rel="next"`)
		fmt.Fprint(w, `[{"type":"file","path":"model-Q4_K_M-00001-of-00002.gguf","size":20}]`)
	}))
	defer server.Close()
	address = server.URL
	client := New()
	client.BaseURL = server.URL
	for i := 0; i < 2; i++ {
		result, err := client.Repository(context.Background(), "owner/model")
		if err != nil || len(result.Variants) != 1 || len(result.Variants[0].Files) != 2 {
			t.Fatalf("Incomplete paginated result: %#v %v", result, err)
		}
	}
}
