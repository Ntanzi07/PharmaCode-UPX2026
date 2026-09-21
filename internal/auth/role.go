package auth

// Role é o papel do usuário no painel. Cada papel inclui as permissões dos
// anteriores: editor < reviewer < admin.
type Role string

const (
	RoleEditor   Role = "editor"   // cadastra e edita remédios, embalagens e bulas
	RoleReviewer Role = "reviewer" // + marca bula como revisada (farmacêutico)
	RoleAdmin    Role = "admin"    // + gerencia usuários
)

var roleLevel = map[Role]int{
	RoleEditor:   1,
	RoleReviewer: 2,
	RoleAdmin:    3,
}

// Valid diz se o papel é um dos conhecidos.
func (r Role) Valid() bool {
	_, ok := roleLevel[r]
	return ok
}

// AtLeast diz se o papel tem, no mínimo, as permissões de min.
func (r Role) AtLeast(min Role) bool {
	return r.Valid() && roleLevel[r] >= roleLevel[min]
}
