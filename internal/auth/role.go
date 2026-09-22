package auth

// Role is the user's role in the admin panel. Each role includes the permissions of the
// previous ones: editor < reviewer < admin.
type Role string

const (
	RoleEditor   Role = "editor"   // creates and edits drugs, packages and leaflets
	RoleReviewer Role = "reviewer" // + marks leaflets as reviewed (pharmacist)
	RoleAdmin    Role = "admin"    // + manages users
)

var roleLevel = map[Role]int{
	RoleEditor:   1,
	RoleReviewer: 2,
	RoleAdmin:    3,
}

// Valid reports whether the role is a known one.
func (r Role) Valid() bool {
	_, ok := roleLevel[r]
	return ok
}

// AtLeast reports whether the role has at least the permissions of min.
func (r Role) AtLeast(min Role) bool {
	return r.Valid() && roleLevel[r] >= roleLevel[min]
}
