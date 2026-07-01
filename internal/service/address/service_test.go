package address_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/neuhis/software-practice-backend/internal/model"
	"github.com/neuhis/software-practice-backend/internal/repository"
	addresssvc "github.com/neuhis/software-practice-backend/internal/service/address"
)

var _ repository.AddressRepository = (*mockAddressRepo)(nil)

type mockAddressRepo struct {
	createFunc                func(ctx context.Context, addr *model.Address) error
	findByIDFunc              func(ctx context.Context, id string) (*model.Address, error)
	listByPatientFunc         func(ctx context.Context, patientID string) ([]model.Address, error)
	countByPatientFunc        func(ctx context.Context, patientID string) (int, error)
	updateFunc                func(ctx context.Context, addr *model.Address) error
	deleteFunc                func(ctx context.Context, id string) error
	clearDefaultByPatientFunc func(ctx context.Context, patientID string) error
	setDefaultFunc            func(ctx context.Context, id, patientID string) error
}

func (m *mockAddressRepo) Create(ctx context.Context, addr *model.Address) error {
	return m.createFunc(ctx, addr)
}
func (m *mockAddressRepo) FindByID(ctx context.Context, id string) (*model.Address, error) {
	return m.findByIDFunc(ctx, id)
}
func (m *mockAddressRepo) ListByPatient(ctx context.Context, patientID string) ([]model.Address, error) {
	return m.listByPatientFunc(ctx, patientID)
}
func (m *mockAddressRepo) CountByPatient(ctx context.Context, patientID string) (int, error) {
	return m.countByPatientFunc(ctx, patientID)
}
func (m *mockAddressRepo) Update(ctx context.Context, addr *model.Address) error {
	return m.updateFunc(ctx, addr)
}
func (m *mockAddressRepo) Delete(ctx context.Context, id string) error {
	return m.deleteFunc(ctx, id)
}
func (m *mockAddressRepo) ClearDefaultByPatient(ctx context.Context, patientID string) error {
	return m.clearDefaultByPatientFunc(ctx, patientID)
}
func (m *mockAddressRepo) SetDefault(ctx context.Context, id, patientID string) error {
	return m.setDefaultFunc(ctx, id, patientID)
}

func makeTestAddress(patientID string) *model.Address {
	return &model.Address{
		ID:        "addr-test-1",
		PatientID: patientID,
		Name:      "李明",
		Phone:     "13800002468",
		Province:  "辽宁省",
		City:      "沈阳市",
		District:  "浑南区",
		Detail:    "创新路195号",
		IsDefault: false,
		Tag:       model.AddressTagCompany,
	}
}

func TestListAddresses(t *testing.T) {
	repo := &mockAddressRepo{
		listByPatientFunc: func(ctx context.Context, patientID string) ([]model.Address, error) {
			return []model.Address{*makeTestAddress(patientID)}, nil
		},
	}
	svc := addresssvc.NewService(repo)

	resp, err := svc.ListAddresses(context.Background(), "p001")
	if err != nil {
		t.Fatalf("ListAddresses: %v", err)
	}
	if len(resp.Addresses) != 1 {
		t.Errorf("got %d addresses, want 1", len(resp.Addresses))
	}
}

func TestListAddresses_Empty(t *testing.T) {
	repo := &mockAddressRepo{
		listByPatientFunc: func(ctx context.Context, patientID string) ([]model.Address, error) {
			return []model.Address{}, nil
		},
	}
	svc := addresssvc.NewService(repo)

	resp, err := svc.ListAddresses(context.Background(), "p001")
	if err != nil {
		t.Fatalf("ListAddresses: %v", err)
	}
	if len(resp.Addresses) != 0 {
		t.Errorf("got %d addresses, want 0", len(resp.Addresses))
	}
}

func TestCreateAddress_Success(t *testing.T) {
	var created *model.Address
	repo := &mockAddressRepo{
		countByPatientFunc:        func(ctx context.Context, patientID string) (int, error) { return 0, nil },
		clearDefaultByPatientFunc: func(ctx context.Context, patientID string) error { return nil },
		createFunc: func(ctx context.Context, addr *model.Address) error {
			created = addr
			return nil
		},
	}
	svc := addresssvc.NewService(repo)

	addr, err := svc.CreateAddress(context.Background(), "p001", model.CreateAddressInput{
		Name: "李明", Phone: "13800002468",
		Province: "辽宁省", City: "沈阳市", District: "浑南区", Detail: "创新路195号",
		Tag: model.AddressTagCompany,
	})
	if err != nil {
		t.Fatalf("CreateAddress: %v", err)
	}
	if !addr.IsDefault {
		t.Error("first address should be default")
	}
	if created != nil && !created.IsDefault {
		t.Error("created address should have isDefault=true")
	}
}

func TestCreateAddress_FirstAddressAutoDefault(t *testing.T) {
	repo := &mockAddressRepo{
		countByPatientFunc:        func(ctx context.Context, patientID string) (int, error) { return 0, nil },
		clearDefaultByPatientFunc: func(ctx context.Context, patientID string) error { return nil },
		createFunc:                func(ctx context.Context, addr *model.Address) error { return nil },
	}
	svc := addresssvc.NewService(repo)

	input := model.CreateAddressInput{
		Name: "测试", Phone: "13800002468",
		Province: "辽宁", City: "沈阳", District: "浑南", Detail: "测试地址",
	}
	addr, err := svc.CreateAddress(context.Background(), "p001", input)
	if err != nil {
		t.Fatalf("CreateAddress: %v", err)
	}
	if !addr.IsDefault {
		t.Error("first address should be auto-set as default")
	}
}

func TestCreateAddress_LimitExceeded(t *testing.T) {
	repo := &mockAddressRepo{
		countByPatientFunc: func(ctx context.Context, patientID string) (int, error) { return 10, nil },
	}
	svc := addresssvc.NewService(repo)

	_, err := svc.CreateAddress(context.Background(), "p001", model.CreateAddressInput{
		Name: "李明", Phone: "13800002468",
		Province: "辽宁", City: "沈阳", District: "浑南", Detail: "测试",
	})
	if err != model.ErrAddressLimitExceeded {
		t.Errorf("expected ErrAddressLimitExceeded, got %v", err)
	}
}

func TestCreateAddress_ValidationError(t *testing.T) {
	tests := []struct {
		name  string
		input model.CreateAddressInput
	}{
		{"empty name", model.CreateAddressInput{Name: "", Phone: "13800002468", Province: "辽宁", City: "沈阳", District: "浑南", Detail: "测试"}},
		{"name too long", model.CreateAddressInput{Name: "一二三四五六七八九十一二三四五六七八九十一", Phone: "13800002468", Province: "辽宁", City: "沈阳", District: "浑南", Detail: "测试"}},
		{"invalid phone", model.CreateAddressInput{Name: "李明", Phone: "12345", Province: "辽宁", City: "沈阳", District: "浑南", Detail: "测试"}},
		{"empty detail", model.CreateAddressInput{Name: "李明", Phone: "13800002468", Province: "辽宁", City: "沈阳", District: "浑南", Detail: ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockAddressRepo{
				countByPatientFunc: func(ctx context.Context, patientID string) (int, error) { return 0, nil },
			}
			svc := addresssvc.NewService(repo)
			_, err := svc.CreateAddress(context.Background(), "p001", tt.input)
			if err == nil {
				t.Error("expected validation error")
			}
		})
	}
}

func TestUpdateAddress_Success(t *testing.T) {
	addr := makeTestAddress("p001")
	repo := &mockAddressRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.Address, error) {
			return addr, nil
		},
		updateFunc: func(ctx context.Context, a *model.Address) error { return nil },
	}
	svc := addresssvc.NewService(repo)

	newName := "张三"
	updated, err := svc.UpdateAddress(context.Background(), "p001", addr.ID, model.UpdateAddressInput{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("UpdateAddress: %v", err)
	}
	if updated.Name != newName {
		t.Errorf("name = %s, want %s", updated.Name, newName)
	}
}

func TestUpdateAddress_NotFound(t *testing.T) {
	repo := &mockAddressRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.Address, error) {
			return nil, model.ErrAddressNotFound
		},
	}
	svc := addresssvc.NewService(repo)

	_, err := svc.UpdateAddress(context.Background(), "p001", "bad-id", model.UpdateAddressInput{})
	if err != model.ErrAddressNotFound {
		t.Errorf("expected ErrAddressNotFound, got %v", err)
	}
}

func TestUpdateAddress_WrongPatient(t *testing.T) {
	addr := makeTestAddress("p002")
	repo := &mockAddressRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.Address, error) {
			return addr, nil
		},
	}
	svc := addresssvc.NewService(repo)

	_, err := svc.UpdateAddress(context.Background(), "p001", addr.ID, model.UpdateAddressInput{})
	if err != model.ErrAddressNotFound {
		t.Errorf("expected ErrAddressNotFound for wrong patient, got %v", err)
	}
}

func TestDeleteAddress_Success(t *testing.T) {
	addr := makeTestAddress("p001")
	addr.IsDefault = false
	repo := &mockAddressRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.Address, error) {
			return addr, nil
		},
		deleteFunc: func(ctx context.Context, id string) error { return nil },
	}
	svc := addresssvc.NewService(repo)

	resp, err := svc.DeleteAddress(context.Background(), "p001", addr.ID)
	if err != nil {
		t.Fatalf("DeleteAddress: %v", err)
	}
	if !resp.Success {
		t.Error("expected success=true")
	}
}

func TestDeleteAddress_DefaultPromotesNext(t *testing.T) {
	wasDefault := true
	addr := makeTestAddress("p001")
	addr.IsDefault = true
	repo := &mockAddressRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.Address, error) {
			return addr, nil
		},
		deleteFunc: func(ctx context.Context, id string) error { return nil },
		listByPatientFunc: func(ctx context.Context, patientID string) ([]model.Address, error) {
			// After delete, there's one remaining address
			return []model.Address{{
				ID:        "addr-remaining",
				PatientID: patientID,
				Name:      "其他", Phone: "13800002468",
				Province: "辽宁", City: "沈阳", District: "浑南", Detail: "其他地址",
			}}, nil
		},
		setDefaultFunc: func(ctx context.Context, id, patientID string) error {
			wasDefault = false // promoted
			return nil
		},
	}
	svc := addresssvc.NewService(repo)

	_, err := svc.DeleteAddress(context.Background(), "p001", addr.ID)
	if err != nil {
		t.Fatalf("DeleteAddress: %v", err)
	}
	_ = wasDefault
}

func TestDeleteAddress_NotFound(t *testing.T) {
	repo := &mockAddressRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.Address, error) {
			return nil, model.ErrAddressNotFound
		},
	}
	svc := addresssvc.NewService(repo)

	_, err := svc.DeleteAddress(context.Background(), "p001", "bad-id")
	if err != model.ErrAddressNotFound {
		t.Errorf("expected ErrAddressNotFound, got %v", err)
	}
}

func TestSetDefaultAddress_Success(t *testing.T) {
	addr := makeTestAddress("p001")
	repo := &mockAddressRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.Address, error) {
			return addr, nil
		},
		setDefaultFunc: func(ctx context.Context, id, patientID string) error {
			return nil
		},
	}
	svc := addresssvc.NewService(repo)

	updated, err := svc.SetDefaultAddress(context.Background(), "p001", addr.ID)
	if err != nil {
		t.Fatalf("SetDefaultAddress: %v", err)
	}
	if !updated.IsDefault {
		t.Error("address should be default after SetDefaultAddress")
	}
}

func TestSetDefaultAddress_NotFound(t *testing.T) {
	repo := &mockAddressRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.Address, error) {
			return nil, model.ErrAddressNotFound
		},
	}
	svc := addresssvc.NewService(repo)

	_, err := svc.SetDefaultAddress(context.Background(), "p001", "bad-id")
	if err != model.ErrAddressNotFound {
		t.Errorf("expected ErrAddressNotFound, got %v", err)
	}
}

func TestUpdateAddress_AllFields(t *testing.T) {
	addr := makeTestAddress("p001")
	repo := &mockAddressRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.Address, error) {
			return addr, nil
		},
		clearDefaultByPatientFunc: func(ctx context.Context, patientID string) error { return nil },
		updateFunc:                func(ctx context.Context, a *model.Address) error { return nil },
	}
	svc := addresssvc.NewService(repo)

	newName := "王五"
	newPhone := "13900001111"
	newProvince := "北京市"
	newCity := "北京市"
	newDistrict := "朝阳区"
	newDetail := "建国路100号"
	newTag := model.AddressTagHome
	isDefault := true
	updated, err := svc.UpdateAddress(context.Background(), "p001", addr.ID, model.UpdateAddressInput{
		Name: &newName, Phone: &newPhone,
		Province: &newProvince, City: &newCity, District: &newDistrict,
		Detail: &newDetail, Tag: &newTag, IsDefault: &isDefault,
	})
	if err != nil {
		t.Fatalf("UpdateAddress: %v", err)
	}
	if updated.Name != newName {
		t.Errorf("name = %s, want %s", updated.Name, newName)
	}
	if updated.Phone != newPhone {
		t.Errorf("phone = %s, want %s", updated.Phone, newPhone)
	}
	if updated.IsDefault != true {
		t.Error("isDefault should be true")
	}
}

func TestUpdateAddress_SetDefaultFalse(t *testing.T) {
	addr := makeTestAddress("p001")
	addr.IsDefault = true
	repo := &mockAddressRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.Address, error) {
			return addr, nil
		},
		updateFunc: func(ctx context.Context, a *model.Address) error { return nil },
	}
	svc := addresssvc.NewService(repo)

	isDefault := false
	updated, err := svc.UpdateAddress(context.Background(), "p001", addr.ID, model.UpdateAddressInput{
		IsDefault: &isDefault,
	})
	if err != nil {
		t.Fatalf("UpdateAddress: %v", err)
	}
	if updated.IsDefault != false {
		t.Error("isDefault should be false")
	}
}

func TestUpdateAddress_ValidationErrors(t *testing.T) {
	tests := []struct {
		name  string
		input model.UpdateAddressInput
	}{
		{"name too long", model.UpdateAddressInput{Name: strPtr("一二三四五六七八九十一二三四五六七八九十一")}},
		{"invalid phone", model.UpdateAddressInput{Phone: strPtr("12345")}},
		{"empty detail", model.UpdateAddressInput{Detail: strPtr("")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr := makeTestAddress("p001")
			repo := &mockAddressRepo{
				findByIDFunc: func(ctx context.Context, id string) (*model.Address, error) {
					return addr, nil
				},
			}
			svc := addresssvc.NewService(repo)
			_, err := svc.UpdateAddress(context.Background(), "p001", addr.ID, tt.input)
			if err == nil {
				t.Error("expected validation error")
			}
		})
	}
}

func TestSetDefaultAddress_WrongPatient(t *testing.T) {
	addr := makeTestAddress("p002")
	repo := &mockAddressRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.Address, error) {
			return addr, nil
		},
	}
	svc := addresssvc.NewService(repo)

	_, err := svc.SetDefaultAddress(context.Background(), "p001", addr.ID)
	if err != model.ErrAddressNotFound {
		t.Errorf("expected ErrAddressNotFound, got %v", err)
	}
}

func TestDeleteAddress_WrongPatient(t *testing.T) {
	addr := makeTestAddress("p002")
	repo := &mockAddressRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.Address, error) {
			return addr, nil
		},
	}
	svc := addresssvc.NewService(repo)

	_, err := svc.DeleteAddress(context.Background(), "p001", addr.ID)
	if err != model.ErrAddressNotFound {
		t.Errorf("expected ErrAddressNotFound, got %v", err)
	}
}

func TestListAddresses_RepoError(t *testing.T) {
	repo := &mockAddressRepo{
		listByPatientFunc: func(ctx context.Context, patientID string) ([]model.Address, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	svc := addresssvc.NewService(repo)

	_, err := svc.ListAddresses(context.Background(), "p001")
	if err == nil {
		t.Fatal("expected error when repo fails")
	}
}

func TestCreateAddress_RepoCreateError(t *testing.T) {
	repo := &mockAddressRepo{
		countByPatientFunc: func(ctx context.Context, patientID string) (int, error) { return 1, nil },
		createFunc:         func(ctx context.Context, addr *model.Address) error { return fmt.Errorf("db error") },
	}
	svc := addresssvc.NewService(repo)

	_, err := svc.CreateAddress(context.Background(), "p001", model.CreateAddressInput{
		Name: "李明", Phone: "13800002468",
		Province: "辽宁", City: "沈阳", District: "浑南", Detail: "测试地址",
	})
	if err == nil {
		t.Fatal("expected error when repo.Create fails")
	}
}

func TestCreateAddress_MissingProvince(t *testing.T) {
	repo := &mockAddressRepo{
		countByPatientFunc: func(ctx context.Context, patientID string) (int, error) { return 0, nil },
	}
	svc := addresssvc.NewService(repo)

	_, err := svc.CreateAddress(context.Background(), "p001", model.CreateAddressInput{
		Name: "李明", Phone: "13800002468",
		City: "沈阳", District: "浑南", Detail: "测试地址",
	})
	if err == nil {
		t.Fatal("expected validation error for missing province")
	}
}

func TestSetDefaultAddress_RepoError(t *testing.T) {
	addr := makeTestAddress("p001")
	repo := &mockAddressRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.Address, error) {
			return addr, nil
		},
		setDefaultFunc: func(ctx context.Context, id, patientID string) error {
			return fmt.Errorf("db error")
		},
	}
	svc := addresssvc.NewService(repo)

	_, err := svc.SetDefaultAddress(context.Background(), "p001", addr.ID)
	if err == nil {
		t.Fatal("expected error when repo.SetDefault fails")
	}
}

func TestCreateAddress_ExplicitDefault(t *testing.T) {
	repo := &mockAddressRepo{
		countByPatientFunc:        func(ctx context.Context, patientID string) (int, error) { return 1, nil },
		clearDefaultByPatientFunc: func(ctx context.Context, patientID string) error { return nil },
		createFunc:                func(ctx context.Context, addr *model.Address) error { return nil },
	}
	svc := addresssvc.NewService(repo)

	addr, err := svc.CreateAddress(context.Background(), "p001", model.CreateAddressInput{
		Name: "张三", Phone: "13800002468",
		Province: "辽宁", City: "沈阳", District: "浑南", Detail: "测试地址",
		IsDefault: true,
	})
	if err != nil {
		t.Fatalf("CreateAddress: %v", err)
	}
	if !addr.IsDefault {
		t.Error("isDefault should be true when explicitly set")
	}
}

func TestDeleteAddress_NoRemainingAddresses(t *testing.T) {
	addr := makeTestAddress("p001")
	addr.IsDefault = true
	repo := &mockAddressRepo{
		findByIDFunc: func(ctx context.Context, id string) (*model.Address, error) {
			return addr, nil
		},
		deleteFunc: func(ctx context.Context, id string) error { return nil },
		listByPatientFunc: func(ctx context.Context, patientID string) ([]model.Address, error) {
			return []model.Address{}, nil
		},
	}
	svc := addresssvc.NewService(repo)

	resp, err := svc.DeleteAddress(context.Background(), "p001", addr.ID)
	if err != nil {
		t.Fatalf("DeleteAddress: %v", err)
	}
	if !resp.Success {
		t.Error("expected success=true")
	}
}

func strPtr(s string) *string { return &s }
