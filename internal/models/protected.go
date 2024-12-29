package models

import "github.com/google/uuid"

type Protected interface {
	IsOwnedBy(p *uuid.UUID) bool
}
