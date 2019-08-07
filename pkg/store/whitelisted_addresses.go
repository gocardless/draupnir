package store

import (
	"database/sql"

	"github.com/gocardless/draupnir/pkg/models"
	_ "github.com/lib/pq" // used to setup the PG driver
)

type WhitelistedAddressStore interface {
	Create(models.WhitelistedAddress) (models.WhitelistedAddress, error)
	List() ([]models.WhitelistedAddress, error)
}

type DBWhitelistedAddressStore struct {
	DB             *sql.DB
	PublicHostname string
}

func (s DBWhitelistedAddressStore) Create(address models.WhitelistedAddress) (models.WhitelistedAddress, error) {
	row := s.DB.QueryRow(
		`INSERT INTO whitelisted_addresses (ip_address, instance_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (ip_address, instance_id) DO UPDATE SET updated_at = NOW()
		 RETURNING updated_at`,
		address.IPAddress,
		address.Instance.ID,
		address.CreatedAt,
		address.UpdatedAt,
	)

	err := row.Scan(&address.UpdatedAt)

	return address, err
}

func (s DBWhitelistedAddressStore) List() ([]models.WhitelistedAddress, error) {
	addresses := make([]models.WhitelistedAddress, 0)

	rows, err := s.DB.Query(
		`SELECT
		   whitelisted_addresses.ip_address,
		   whitelisted_addresses.created_at,
		   whitelisted_addresses.updated_at,
		   instances.id AS instance_id,
		   instances.port AS instance_port,
		   instances.user_email AS instance_user_email
		 FROM whitelisted_addresses
		 JOIN instances ON instances.id = whitelisted_addresses.instance_id
		 ORDER BY whitelisted_addresses.created_at ASC`,
	)
	if err != nil {
		return addresses, err
	}

	defer rows.Close()

	for rows.Next() {
		var address models.WhitelistedAddress
		// The instance struct is only populated with the 3 fields that we need: ID,
		// port and user email. Other fields will be left at their 'zero value', but
		// ignored in the code that calls this.
		var instance models.Instance

		err = rows.Scan(
			&address.IPAddress,
			&address.CreatedAt,
			&address.UpdatedAt,
			&instance.ID,
			&instance.Port,
			&instance.UserEmail,
		)

		if err != nil {
			return nil, err
		}

		address.Instance = &instance

		addresses = append(addresses, address)
	}

	return addresses, nil
}
