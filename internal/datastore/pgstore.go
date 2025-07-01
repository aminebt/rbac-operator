package datastore

import (
	"database/sql"
	"fmt"

	"github.com/aminebt/rbac-operator/rbac"
	_ "github.com/lib/pq"
)

// TBD - read from configfile and env var (password)
const (
	host     = "10.227.54.173"
	port     = 31432
	user     = "rbacdbuser"
	password = "password"
	dbname   = "rbacdb"
)

type PostgresDataStore struct {
	Db *sql.DB
}

type pgGroup struct {
	id          int
	description string
	rbac.Group
}

func (pgds *PostgresDataStore) CleanUp() {
	pgds.Db.Close()
}

func NewPostgresDataStore() (DataStore, error) {
	connString := fmt.Sprintf("host=%s port=%d user=%s password=%s database=%s sslmode=require", host, port, user, password, dbname)

	// simply validates the arguments
	db, err := sql.Open("postgres", connString)

	if err != nil {
		return nil, err
	}

	pgds := &PostgresDataStore{
		Db: db,
	}

	return pgds, nil
}

func (pgds *PostgresDataStore) GetGroup(groupName string) (rbac.Group, int, error) {
	sqlStatement := `SELECT * FROM user_groups WHERE group_name=$1;`
	var gr pgGroup
	row := pgds.Db.QueryRow(sqlStatement, groupName)
	err := row.Scan(&gr.id, &gr.Name, &gr.description)
	switch err {
	// must define a generic error ("Not Found")
	case sql.ErrNoRows:
		fmt.Println("No rows were returned!")
		return rbac.Group{}, 0, err
	case nil:
		return gr.Group, gr.id, nil
	default:
		return rbac.Group{}, 0, err
	}
}

func (pgds *PostgresDataStore) CreateGroup(gr rbac.Group) (int, error) {
	// sqlStatement := `
	// INSERT INTO user_groups (group_name, description)
	// VALUES ($1, $2)
	// ON CONFLICT (group_name) DO NOTHING
	// `

	sqlStatement := `
	WITH inserted AS (
		INSERT INTO user_groups (group_name, description)
		VALUES ($1, $2)
		ON CONFLICT (group_name) DO NOTHING
		RETURNING id
	)
	SELECT id FROM inserted
	UNION ALL
	SELECT id FROM user_groups WHERE group_name = $1 AND NOT EXISTS (SELECT 1 FROM inserted);	
	`

	var id int
	err := pgds.Db.QueryRow(sqlStatement, gr.Name, "placeholder description").Scan(&id)
	if err != nil {
		return 0, err
	}
	// TBD - how to return the id
	return id, nil
}

func (pgds *PostgresDataStore) DeleteGroup(groupName string) (rbac.Group, error) {
	return rbac.Group{}, nil
}

// Role methods
func (pgds *PostgresDataStore) GetRole(roleName string) (rbac.Role, error) {
	return rbac.Role{}, nil
}

func (pgds *PostgresDataStore) CreateUpdateRole(roleName string) (rbac.Role, error) {
	return rbac.Role{}, nil
}

func (pgds *PostgresDataStore) DeleteRole(roleName string) (rbac.Role, error) {
	return rbac.Role{}, nil
}

// GRBinding
func (pgds *PostgresDataStore) GetGRBinding(roleName string) (rbac.GroupRoleBinding, error) {
	return rbac.GroupRoleBinding{}, nil
}

func (pgds *PostgresDataStore) CreateGRBinding(roleName string) (rbac.GroupRoleBinding, error) {
	return rbac.GroupRoleBinding{}, nil
}

func (pgds *PostgresDataStore) DeleteGRBinding(roleName string) (rbac.GroupRoleBinding, error) {
	return rbac.GroupRoleBinding{}, nil
}
