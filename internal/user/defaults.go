package user

import "github.com/google/uuid"

// wellKnownMemberRoleID is the UUID used for the seeded "member" role.
// This must match the UUID inserted by the initial migration seed.
var wellKnownMemberRoleID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// DefaultRoleID returns the pre-seeded default role ID for new users.
// The value is fixed in migrations/000001_init.up.sql.
func DefaultRoleID() uuid.UUID {
	return wellKnownMemberRoleID
}
