package api_test

import (
	"encoding/json"
	"testing"

	"github.com/neuhis/software-practice-backend/pkg/api"
)

func TestSuccessResponse(t *testing.T) {
	resp := api.SuccessResponse("test data")

	if !resp.Success {
		t.Errorf("SuccessResponse.Success = false, want true")
	}
	if resp.Data == nil {
		t.Fatal("SuccessResponse.Data = nil, want non-nil")
	}
	if *resp.Data != "test data" {
		t.Errorf("SuccessResponse.Data = %v, want %v", *resp.Data, "test data")
	}
	if resp.Error != nil {
		t.Errorf("SuccessResponse.Error = %v, want nil", resp.Error)
	}

	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal(SuccessResponse) failed: %v", err)
	}

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatalf("json.Unmarshal(SuccessResponse) failed: %v", err)
	}

	var success bool
	if err := json.Unmarshal(parsed["success"], &success); err != nil {
		t.Fatalf("unmarshal success field: %v", err)
	}
	if !success {
		t.Error("JSON success = false, want true")
	}

	var data string
	if err := json.Unmarshal(parsed["data"], &data); err != nil {
		t.Fatalf("unmarshal data field: %v", err)
	}
	if data != "test data" {
		t.Errorf("JSON data = %q, want %q", data, "test data")
	}

	if parsed["error"] == nil {
		t.Fatal("JSON error field missing")
	}
	var errVal interface{}
	if err := json.Unmarshal(parsed["error"], &errVal); err != nil {
		t.Fatalf("unmarshal error field: %v", err)
	}
	if errVal != nil {
		t.Errorf("JSON error = %v, want nil", errVal)
	}
}

func TestErrorResponse(t *testing.T) {
	resp := api.ErrorResponse("something went wrong")

	if resp.Success {
		t.Errorf("ErrorResponse.Success = true, want false")
	}
	if resp.Data != nil {
		t.Errorf("ErrorResponse.Data = %v, want nil", resp.Data)
	}
	if resp.Error == nil {
		t.Fatal("ErrorResponse.Error = nil, want non-nil")
	}
	if resp.Error != "something went wrong" {
		t.Errorf("ErrorResponse.Error = %v, want %v", resp.Error, "something went wrong")
	}

	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal(ErrorResponse) failed: %v", err)
	}

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatalf("json.Unmarshal(ErrorResponse) failed: %v", err)
	}

	var success bool
	if err := json.Unmarshal(parsed["success"], &success); err != nil {
		t.Fatalf("unmarshal success field: %v", err)
	}
	if success {
		t.Error("JSON success = true, want false")
	}

	if parsed["data"] == nil {
		t.Fatal("JSON data field missing")
	}
	var dataVal interface{}
	if err := json.Unmarshal(parsed["data"], &dataVal); err != nil {
		t.Fatalf("unmarshal data field: %v", err)
	}
	if dataVal != nil {
		t.Errorf("JSON data = %v, want nil", dataVal)
	}

	var errVal string
	if err := json.Unmarshal(parsed["error"], &errVal); err != nil {
		t.Fatalf("unmarshal error field: %v", err)
	}
	if errVal != "something went wrong" {
		t.Errorf("JSON error = %q, want %q", errVal, "something went wrong")
	}
}

func TestPageResult(t *testing.T) {
	cursor := "next-page-token"
	items := []int{1, 2, 3}
	page := api.NewPageResult(items, &cursor, true)

	if len(page.Items) != 3 {
		t.Errorf("page.Items length = %d, want 3", len(page.Items))
	}
	if page.Items[0] != 1 || page.Items[1] != 2 || page.Items[2] != 3 {
		t.Errorf("page.Items = %v, want [1 2 3]", page.Items)
	}
	if page.NextCursor == nil {
		t.Fatal("page.NextCursor = nil, want non-nil")
	}
	if *page.NextCursor != "next-page-token" {
		t.Errorf("page.NextCursor = %q, want %q", *page.NextCursor, "next-page-token")
	}
	if !page.HasMore {
		t.Errorf("page.HasMore = false, want true")
	}

	b, err := json.Marshal(page)
	if err != nil {
		t.Fatalf("json.Marshal(PageResult) failed: %v", err)
	}

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatalf("json.Unmarshal(PageResult) failed: %v", err)
	}

	if _, ok := parsed["nextCursor"]; !ok {
		t.Error("JSON nextCursor field missing, want present")
	}

	var hasMore bool
	if err := json.Unmarshal(parsed["hasMore"], &hasMore); err != nil {
		t.Fatalf("unmarshal hasMore field: %v", err)
	}
	if !hasMore {
		t.Error("JSON hasMore = false, want true")
	}
}

func TestPageResultEmpty(t *testing.T) {
	page := api.NewPageResult([]string{}, nil, false)

	if page.Items == nil {
		t.Errorf("page.Items = nil, want []")
	}
	if len(page.Items) != 0 {
		t.Errorf("page.Items length = %d, want 0", len(page.Items))
	}
	if page.NextCursor != nil {
		t.Errorf("page.NextCursor = %v, want nil", page.NextCursor)
	}
	if page.HasMore {
		t.Errorf("page.HasMore = true, want false")
	}

	b, err := json.Marshal(page)
	if err != nil {
		t.Fatalf("json.Marshal(empty PageResult) failed: %v", err)
	}

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatalf("json.Unmarshal(empty PageResult) failed: %v", err)
	}

	var items []interface{}
	if err := json.Unmarshal(parsed["items"], &items); err != nil {
		t.Fatalf("unmarshal items field: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("JSON items length = %d, want 0", len(items))
	}

	if _, ok := parsed["nextCursor"]; ok {
		t.Error("JSON nextCursor present, want omitted for nil")
	}

	var hasMore bool
	if err := json.Unmarshal(parsed["hasMore"], &hasMore); err != nil {
		t.Fatalf("unmarshal hasMore field: %v", err)
	}
	if hasMore {
		t.Error("JSON hasMore = true, want false")
	}
}

func TestSuccessResponseWithMeta(t *testing.T) {
	type meta struct {
		Total int `json:"total"`
	}
	resp := api.SuccessResponseWithMeta("data", meta{Total: 100})

	if !resp.Success {
		t.Error("SuccessResponseWithMeta.Success = false, want true")
	}
	if resp.Data == nil || *resp.Data != "data" {
		t.Errorf("SuccessResponseWithMeta.Data = %v, want \"data\"", resp.Data)
	}
	if resp.Meta == nil {
		t.Fatal("SuccessResponseWithMeta.Meta = nil, want non-nil")
	}

	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if _, ok := parsed["meta"]; !ok {
		t.Error("JSON meta field missing")
	}
}

func TestCursorFromQuery(t *testing.T) {
	t.Run("empty string returns nil", func(t *testing.T) {
		result := api.CursorFromQuery("")
		if result != nil {
			t.Errorf("CursorFromQuery(\"\") = %v, want nil", result)
		}
	})

	t.Run("non-empty string returns pointer", func(t *testing.T) {
		result := api.CursorFromQuery("abc123")
		if result == nil {
			t.Fatal("CursorFromQuery(\"abc123\") = nil, want non-nil")
		}
		if *result != "abc123" {
			t.Errorf("CursorFromQuery(\"abc123\") = %q, want %q", *result, "abc123")
		}
	})

	t.Run("whitespace string returns pointer", func(t *testing.T) {
		result := api.CursorFromQuery("   ")
		if result == nil {
			t.Fatal("CursorFromQuery(\"   \") = nil, want non-nil")
		}
		if *result != "   " {
			t.Errorf("CursorFromQuery(\"   \") = %q, want %q", *result, "   ")
		}
	})
}
