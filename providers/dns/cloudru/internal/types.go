package internal

import (
	"fmt"
	"time"
)

const TxtRecordType = "PUBLIC_RECORD_MANAGED_TYPE_TXT"

type Token struct {
	// The bearer token for use in API requests
	AccessToken string `json:"access_token"`
	TokenID     string `json:"id_token"`
	TokenType   string `json:"token_type"`
	// Number in seconds before the expiration
	ExpiresIn       int    `json:"expires_in"`
	NotBeforePolicy int    `json:"not-before-policy"`
	Scope           string `json:"scope"`

	Deadline time.Time `json:"-"`
}

type authResponseError struct {
	ErrorMsg         string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func (a authResponseError) Error() string {
	return fmt.Sprintf("%s: %s", a.ErrorMsg, a.ErrorDescription)
}

type ZonesAPIResponse struct {
	Zones  []Zone `json:"zones"`
	Page   int    `json:"page"`
	Offset int    `json:"offset"`
	Total  int    `json:"total"`
}

type Zone struct {
	Meta         Meta          `json:"meta,omitempty"`
	ProjectID    string        `json:"projectId,omitempty"`
	Name         string        `json:"name,omitempty"`
	Domain       string        `json:"domain,omitempty"`
	Role         string        `json:"role,omitempty"`
	ReadOnly     bool          `json:"readOnly,omitempty"`
	ActiveState  bool          `json:"activeState,omitempty"`
	State        string        `json:"state,omitempty"`
	CountRecords string        `json:"countRecords,omitempty"`
	CountValues  string        `json:"countValues,omitempty"`
	Description  string        `json:"description,omitempty"`
	Tags         []interface{} `json:"tags,omitempty"`
}

type RecordsAPIResponse struct {
	Records []Record `json:"records"`
	Page    int      `json:"page"`
	Offset  int      `json:"offset"`
	Total   int      `json:"total"`
}

type Record struct {
	Meta        Meta          `json:"meta,omitempty"`
	ZoneID      string        `json:"zoneId,omitempty"`
	Name        string        `json:"name,omitempty"`
	Type        string        `json:"type,omitempty"`
	Values      []string      `json:"values,omitempty"`
	TTL         int           `json:"ttl,omitempty"`
	ReadOnly    bool          `json:"readOnly,omitempty"`
	Description string        `json:"description,omitempty"`
	Tags        []interface{} `json:"tags,omitempty"`
	State       string        `json:"state,omitempty"`
}

type CreateRecordRequest struct {
	Type   string   `json:"type"`
	Name   string   `json:"name"`
	TTL    int      `json:"ttl"`
	Values []string `json:"values"`
	ZoneID string   `json:"zoneId"`
}

type Meta struct {
	ID        string    `json:"id,omitempty"`
	TaskID    string    `json:"taskId,omitempty"`
	CreateAt  time.Time `json:"createAt,omitempty"`
	UpdatedAt time.Time `json:"updatedAtv"`
}
