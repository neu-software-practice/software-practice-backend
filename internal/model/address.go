package model

import "time"

// Address represents a patient's delivery address.
type Address struct {
	ID        string     `json:"id"`
	PatientID string     `json:"patientId"`
	Name      string     `json:"name"`
	Phone     string     `json:"phone"`
	Province  string     `json:"province"`
	City      string     `json:"city"`
	District  string     `json:"district"`
	Detail    string     `json:"detail"`
	IsDefault bool       `json:"isDefault"`
	Tag       AddressTag `json:"tag"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

// CreateAddressInput is the request body for POST /patients/:patientId/addresses.
type CreateAddressInput struct {
	PatientID string     `json:"patientId"`
	Name      string     `json:"name"`
	Phone     string     `json:"phone"`
	Province  string     `json:"province"`
	City      string     `json:"city"`
	District  string     `json:"district"`
	Detail    string     `json:"detail"`
	IsDefault bool       `json:"isDefault"`
	Tag       AddressTag `json:"tag" binding:"max=20"`
}

// UpdateAddressInput is the request body for PATCH /patients/:patientId/addresses/:addressId.
// All fields are optional; only non-nil pointer fields are applied.
type UpdateAddressInput struct {
	Name      *string     `json:"name,omitempty"`
	Phone     *string     `json:"phone,omitempty"`
	Province  *string     `json:"province,omitempty"`
	City      *string     `json:"city,omitempty"`
	District  *string     `json:"district,omitempty"`
	Detail    *string     `json:"detail,omitempty"`
	IsDefault *bool       `json:"isDefault,omitempty"`
	Tag       *AddressTag `json:"tag,omitempty" binding:"omitempty,max=20"`
}

// AddressListResponse is the response for GET /patients/:patientId/addresses.
type AddressListResponse struct {
	Addresses []Address `json:"addresses"`
}

// DeleteAddressResponse is the response for DELETE /patients/:patientId/addresses/:addressId.
type DeleteAddressResponse struct {
	Success bool `json:"success"`
}
