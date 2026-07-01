package address

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/neuhis/software-practice-backend/internal/model"
	"github.com/neuhis/software-practice-backend/internal/repository"
)

// Service handles address book business logic.
type Service struct {
	addressRepo repository.AddressRepository
}

// NewService creates a new AddressService.
func NewService(addressRepo repository.AddressRepository) *Service {
	return &Service{addressRepo: addressRepo}
}

var (
	phoneRegex = regexp.MustCompile(`^1[3-9]\d{9}$`)
)

// ListAddresses returns all addresses for a patient.
func (s *Service) ListAddresses(ctx context.Context, patientID string) (*model.AddressListResponse, error) {
	addrs, err := s.addressRepo.ListByPatient(ctx, patientID)
	if err != nil {
		return nil, fmt.Errorf("list addresses: %w", err)
	}
	return &model.AddressListResponse{Addresses: addrs}, nil
}

// CreateAddress creates a new delivery address for a patient.
func (s *Service) CreateAddress(ctx context.Context, patientID string, input model.CreateAddressInput) (*model.Address, error) {
	// Validate
	if err := validateCreateAddressInput(input); err != nil {
		return nil, err
	}

	// Check limit
	count, err := s.addressRepo.CountByPatient(ctx, patientID)
	if err != nil {
		return nil, fmt.Errorf("count addresses: %w", err)
	}
	if count >= 10 {
		return nil, model.ErrAddressLimitExceeded
	}

	// If this is the first address, auto-set as default
	if count == 0 {
		input.IsDefault = true
	}

	// If setting as default, clear other defaults
	if input.IsDefault {
		if err := s.addressRepo.ClearDefaultByPatient(ctx, patientID); err != nil {
			return nil, fmt.Errorf("clear defaults: %w", err)
		}
	}

	now := time.Now()
	addr := &model.Address{
		ID:        uuid.New().String(),
		PatientID: patientID,
		Name:      input.Name,
		Phone:     input.Phone,
		Province:  input.Province,
		City:      input.City,
		District:  input.District,
		Detail:    input.Detail,
		IsDefault: input.IsDefault,
		Tag:       &input.Tag,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.addressRepo.Create(ctx, addr); err != nil {
		return nil, fmt.Errorf("create address: %w", err)
	}

	return addr, nil
}

// UpdateAddress updates an existing address.
func (s *Service) UpdateAddress(ctx context.Context, patientID, addressID string, input model.UpdateAddressInput) (*model.Address, error) {
	addr, err := s.addressRepo.FindByID(ctx, addressID)
	if err != nil {
		return nil, err
	}

	if addr.PatientID != patientID {
		return nil, model.ErrAddressNotFound
	}

	// Validate optional fields
	if err := validateUpdateAddressInput(input); err != nil {
		return nil, err
	}

	// Build updated address by copying and applying changes (immutable pattern)
	updatedAddr := *addr
	if input.Name != nil {
		updatedAddr.Name = *input.Name
	}
	if input.Phone != nil {
		updatedAddr.Phone = *input.Phone
	}
	if input.Province != nil {
		updatedAddr.Province = *input.Province
	}
	if input.City != nil {
		updatedAddr.City = *input.City
	}
	if input.District != nil {
		updatedAddr.District = *input.District
	}
	if input.Detail != nil {
		updatedAddr.Detail = *input.Detail
	}
	if input.Tag != nil {
		updatedAddr.Tag = input.Tag
	}

	// Handle IsDefault
	if input.IsDefault != nil {
		if *input.IsDefault {
			if err := s.addressRepo.ClearDefaultByPatient(ctx, patientID); err != nil {
				return nil, fmt.Errorf("clear defaults: %w", err)
			}
		}
		updatedAddr.IsDefault = *input.IsDefault
	}

	if err := s.addressRepo.Update(ctx, &updatedAddr); err != nil {
		return nil, fmt.Errorf("update address: %w", err)
	}

	return &updatedAddr, nil
}

// DeleteAddress deletes an address. If the deleted address was the default,
// the first remaining address (if any) is promoted to default.
func (s *Service) DeleteAddress(ctx context.Context, patientID, addressID string) (*model.DeleteAddressResponse, error) {
	addr, err := s.addressRepo.FindByID(ctx, addressID)
	if err != nil {
		return nil, err
	}

	if addr.PatientID != patientID {
		return nil, model.ErrAddressNotFound
	}

	wasDefault := addr.IsDefault

	if err := s.addressRepo.Delete(ctx, addressID); err != nil {
		return nil, fmt.Errorf("delete address: %w", err)
	}

	// If the deleted address was default and other addresses remain, promote the first
	if wasDefault {
		addrs, err := s.addressRepo.ListByPatient(ctx, patientID)
		if err != nil {
			return nil, fmt.Errorf("list remaining addresses: %w", err)
		}
		if len(addrs) > 0 {
			if err := s.addressRepo.SetDefault(ctx, addrs[0].ID, patientID); err != nil {
				return nil, fmt.Errorf("promote new default: %w", err)
			}
		}
	}

	return &model.DeleteAddressResponse{Success: true}, nil
}

// SetDefaultAddress sets an address as the default for a patient.
func (s *Service) SetDefaultAddress(ctx context.Context, patientID, addressID string) (*model.Address, error) {
	addr, err := s.addressRepo.FindByID(ctx, addressID)
	if err != nil {
		return nil, err
	}

	if addr.PatientID != patientID {
		return nil, model.ErrAddressNotFound
	}

	if err := s.addressRepo.SetDefault(ctx, addressID, patientID); err != nil {
		return nil, fmt.Errorf("set default: %w", err)
	}

	addr.IsDefault = true
	return addr, nil
}

// Validation functions

func validateCreateAddressInput(input model.CreateAddressInput) error {
	if len(input.Name) < 1 || len(input.Name) > 20 {
		return fmt.Errorf("%w: name must be 1-20 characters", model.ErrValidation)
	}
	if !phoneRegex.MatchString(input.Phone) {
		return fmt.Errorf("%w: phone must be an 11-digit mainland China mobile number", model.ErrValidation)
	}
	if len(input.Detail) < 1 || len(input.Detail) > 200 {
		return fmt.Errorf("%w: detail must be 1-200 characters", model.ErrValidation)
	}
	if input.Province == "" || input.City == "" || input.District == "" {
		return fmt.Errorf("%w: province, city, and district are required", model.ErrValidation)
	}
	if input.Tag != "" {
		trimmed := strings.TrimSpace(string(input.Tag))
		if len(trimmed) < 1 || len(trimmed) > 20 {
			return fmt.Errorf("%w: tag must be 1-20 characters when provided", model.ErrValidation)
		}
	}
	return nil
}

func validateUpdateAddressInput(input model.UpdateAddressInput) error {
	if input.Name != nil && (len(*input.Name) < 1 || len(*input.Name) > 20) {
		return fmt.Errorf("%w: name must be 1-20 characters", model.ErrValidation)
	}
	if input.Phone != nil && !phoneRegex.MatchString(*input.Phone) {
		return fmt.Errorf("%w: phone must be an 11-digit mainland China mobile number", model.ErrValidation)
	}
	if input.Detail != nil && (len(*input.Detail) < 1 || len(*input.Detail) > 200) {
		return fmt.Errorf("%w: detail must be 1-200 characters", model.ErrValidation)
	}
	if input.Tag != nil {
		trimmed := strings.TrimSpace(string(*input.Tag))
		if len(trimmed) < 1 || len(trimmed) > 20 {
			return fmt.Errorf("%w: tag must be 1-20 characters when provided", model.ErrValidation)
		}
	}
	return nil
}
