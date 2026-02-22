package users

const usersTable = "users"

const (
	usersTableColumnId       = "id"
	usersTableColumnUsername = "username"
	usersTablePasswordHash   = "password_hash"
)

var usersTableColumns = []string{
	usersTableColumnId,
	usersTableColumnUsername,
	usersTablePasswordHash,
}
