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

func TestNewPageResponse(t *testing.T) {
	items := []string{"a", "b", "c"}
	resp := api.NewPageResponse(items, 10, 1, 20)

	if len(resp.Items) != 3 {
		t.Errorf("Items length = %d, want 3", len(resp.Items))
	}
	if resp.Items[0] != "a" {
		t.Errorf("Items[0] = %q, want %q", resp.Items[0], "a")
	}
	if resp.Total != 10 {
		t.Errorf("Total = %d, want 10", resp.Total)
	}
	if resp.Page != 1 {
		t.Errorf("Page = %d, want 1", resp.Page)
	}
	if resp.PageSize != 20 {
		t.Errorf("PageSize = %d, want 20", resp.PageSize)
	}

	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal(PageResponse) failed: %v", err)
	}

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatalf("json.Unmarshal(PageResponse) failed: %v", err)
	}

	if _, ok := parsed["items"]; !ok {
		t.Error("JSON items field missing")
	}
	if _, ok := parsed["total"]; !ok {
		t.Error("JSON total field missing")
	}
	if _, ok := parsed["page"]; !ok {
		t.Error("JSON page field missing")
	}
	if _, ok := parsed["pageSize"]; !ok {
		t.Error("JSON pageSize field missing")
	}
}

func TestNewPageResponse_Empty(t *testing.T) {
	resp := api.NewPageResponse([]int{}, 0, 1, 10)

	if len(resp.Items) != 0 {
		t.Errorf("Items length = %d, want 0", len(resp.Items))
	}
	if resp.Items == nil {
		t.Error("Items should be non-nil empty slice")
	}
	if resp.Total != 0 {
		t.Errorf("Total = %d, want 0", resp.Total)
	}
}

func TestErrorResponse_WithStruct(t *testing.T) {
	type errDetail struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	errResp := api.ErrorResponse(errDetail{Code: "NOT_FOUND", Message: "资源不存在"})

	if errResp.Success {
		t.Error("ErrorResponse.Success = true, want false")
	}
	if errResp.Data != nil {
		t.Errorf("ErrorResponse.Data = %v, want nil", errResp.Data)
	}
	if errResp.Error == nil {
		t.Fatal("ErrorResponse.Error = nil, want non-nil")
	}

	b, err := json.Marshal(errResp)
	if err != nil {
		t.Fatalf("json.Marshal(ErrorResponse with struct) failed: %v", err)
	}

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	// Verify the error field is a JSON object
	var errObj map[string]string
	if err := json.Unmarshal(parsed["error"], &errObj); err != nil {
		t.Fatalf("error field is not a JSON object: %v", err)
	}
	if errObj["code"] != "NOT_FOUND" {
		t.Errorf("error.code = %q, want %q", errObj["code"], "NOT_FOUND")
	}
}

func TestSuccessResponse_WithStruct(t *testing.T) {
	type user struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	u := user{ID: "u1", Name: "测试"}
	resp := api.SuccessResponse(u)

	if !resp.Success {
		t.Error("SuccessResponse.Success = false, want true")
	}
	if resp.Data == nil {
		t.Fatal("SuccessResponse.Data = nil, want non-nil")
	}
	if resp.Data.ID != "u1" {
		t.Errorf("Data.ID = %q, want %q", resp.Data.ID, "u1")
	}

	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal(SuccessResponse with struct) failed: %v", err)
	}

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	var dataObj map[string]string
	if err := json.Unmarshal(parsed["data"], &dataObj); err != nil {
		t.Fatalf("data field is not a JSON object: %v", err)
	}
	if dataObj["id"] != "u1" {
		t.Errorf("data.id = %q, want %q", dataObj["id"], "u1")
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
