package entities

import (
	"ductifact/internal/domain/valueobjects"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrEmptyProjectName = errors.New("project name cannot be empty")
	ErrNilProjectClient = errors.New("project client ID cannot be nil")
)

type Project struct {
	ID          uuid.UUID
	Name        string
	Address     string
	ManagerName string
	Phone       string
	Description string
	ClientID    uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time // nil = active, non-nil = soft-deleted
}

// CreateProjectParams groups all parameters needed to create a Project.
// Required: Name, ClientID. Optional: Address, ManagerName, Phone, Description.
type CreateProjectParams struct {
	Name        string
	Address     string
	ManagerName string
	Phone       string
	Description string
	ClientID    uuid.UUID
}

// UpdateProjectParams groups all fields that can be updated on a Project.
// nil = field not provided (no change), non-nil = new value.
type UpdateProjectParams struct {
	Name        *string
	Address     *string
	ManagerName *string
	Phone       *string
	Description *string
}

// HasChanges returns true if at least one field is set.
func (p UpdateProjectParams) HasChanges() bool {
	return p.Name != nil || p.Address != nil || p.ManagerName != nil ||
		p.Phone != nil || p.Description != nil
}

// NewProject is the only way to create a valid Project.
// It validates all business rules via the setters (single source of truth)
// and returns an error if any fail.
func NewProject(params CreateProjectParams) (*Project, error) {
	if params.ClientID == uuid.Nil {
		return nil, ErrNilProjectClient
	}

	now := time.Now()
	p := &Project{
		ID:        uuid.New(),
		ClientID:  params.ClientID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Validate through setters — each setter is the single source of truth.
	if err := p.SetName(params.Name); err != nil {
		return nil, err
	}
	if err := p.SetAddress(params.Address); err != nil {
		return nil, err
	}
	if err := p.SetManagerName(params.ManagerName); err != nil {
		return nil, err
	}
	if err := p.SetPhone(params.Phone); err != nil {
		return nil, err
	}
	if err := p.SetDescription(params.Description); err != nil {
		return nil, err
	}

	return p, nil
}

// SetName validates and updates the project's name.
func (p *Project) SetName(name string) error {
	if name == "" {
		return ErrEmptyProjectName
	}
	p.Name = name
	return nil
}

// SetAddress updates the project's address.
// An empty address clears the field.
func (p *Project) SetAddress(address string) error {
	p.Address = address
	return nil
}

// SetManagerName updates the project's manager name.
// An empty manager name clears the field.
func (p *Project) SetManagerName(managerName string) error {
	p.ManagerName = managerName
	return nil
}

// SetPhone validates format via the Phone VO and updates the project's phone.
func (p *Project) SetPhone(phone string) error {
	valid, err := valueobjects.NewPhone(phone)
	if err != nil {
		return err
	}
	p.Phone = valid.String()
	return nil
}

// SetDescription validates via the Description VO and updates the project's description.
func (p *Project) SetDescription(description string) error {
	valid, err := valueobjects.NewDescription(description)
	if err != nil {
		return err
	}
	p.Description = valid.String()
	return nil
}
