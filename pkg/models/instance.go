package models

import (
	"time"
)

type Instance struct {
	ID           int    `jsonapi:"primary,instances"`
	Hostname     string `jsonapi:"attr,hostname"`
	ImageID      int    `jsonapi:"attr,image_id"`
	UserEmail    string
	RefreshToken string
	CreatedAt    time.Time `jsonapi:"attr,created_at,iso8601"`
	UpdatedAt    time.Time `jsonapi:"attr,updated_at,iso8601"`
	Port         int       `jsonapi:"attr,port"`

	Credentials *InstanceCredentials `jsonapi:"relation,credentials"`
}

func NewInstance(imageID int, email, refreshToken string) Instance {
	return Instance{
		ImageID:      imageID,
		UserEmail:    email,
		RefreshToken: refreshToken,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

type InstanceCredentials struct {
	// The JSON:API spec says that we should have an ID field, even though we'll
	// just be setting it to the same value as the instance ID.
	// It would be nice to have this struct as an embedded object, rather than a relation, but that's not possible due to these issues:
	// https://github.com/google/jsonapi/issues/74
	// https://github.com/google/jsonapi/issues/117
	ID                int    `jsonapi:"primary,credentials"`
	CACertificate     string `jsonapi:"attr,ca_certificate"`
	ClientCertificate string `jsonapi:"attr,client_certificate"`
	ClientKey         string `jsonapi:"attr,client_key"`
}

func NewInstanceCredentials(id int, caCert string, clientCert string, clientKey string) InstanceCredentials {
	return InstanceCredentials{
		ID:                id,
		CACertificate:     caCert,
		ClientCertificate: clientCert,
		ClientKey:         clientKey,
	}
}
