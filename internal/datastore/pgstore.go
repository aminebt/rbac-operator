package datastore

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/aminebt/rbac-operator/internal/env"
	"github.com/aminebt/rbac-operator/rbac"
	"github.com/lib/pq"
)

// TBD - read from configfile and env var (password)
const (
	host     = "10.227.53.140"
	port     = 31432
	user     = "rbacdbuser"
	password = "password"
	dbname   = "rbacdb"
)

type PostgresDataStore struct {
	Db *sql.DB
}

// TBD include description in rbac.Group
type pgGroup struct {
	id          int
	description string
	rbac.Group
}

// TBD include description in rbac.Role
type pgRole struct {
	id          int
	description string
	rbac.Role
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

func (pgds *PostgresDataStore) GetGroup(env env.Environment, groupName string) (rbac.Group, int, error) {
	tableName := strings.Join([]string{pq.QuoteIdentifier(env.Name), "user_groups"}, ".")
	sqlStatement := fmt.Sprintf(`SELECT * FROM %s WHERE group_name=$1;`, tableName)
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

func (pgds *PostgresDataStore) CreateGroup(env env.Environment, gr rbac.Group) (int, error) {
	tableName := strings.Join([]string{pq.QuoteIdentifier(env.Name), "user_groups"}, ".")

	sqlStatement := fmt.Sprintf(`
	WITH inserted AS (
		INSERT INTO %s (group_name, description)
		VALUES ($1, $2)
		ON CONFLICT (group_name) DO NOTHING
		RETURNING id
	)
	SELECT id FROM inserted
	UNION ALL
	SELECT id FROM %s WHERE group_name = $1 AND NOT EXISTS (SELECT 1 FROM inserted);	
	`, tableName, tableName)

	var id int
	err := pgds.Db.QueryRow(sqlStatement, gr.Name, "placeholder description").Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (pgds *PostgresDataStore) DeleteGroup(env env.Environment, groupName string) (int, error) {
	tableName := strings.Join([]string{pq.QuoteIdentifier(env.Name), "user_groups"}, ".")
	sqlStatement := fmt.Sprintf(`
	DELETE FROM %s
    WHERE group_name = $1 
	RETURNING id;	
	`, tableName)
	var id int
	err := pgds.Db.QueryRow(sqlStatement, groupName).Scan(&id)
	switch err {
	// must define a generic error ("Not Found")
	case sql.ErrNoRows:
		fmt.Println("Group to delete not found")
		return 0, nil
	case nil:
		return id, nil
	default:
		return 0, err
	}
}

// Role methods
func (pgds *PostgresDataStore) GetRole(env env.Environment, roleName string) (rbac.Role, int, error) {
	tableName := strings.Join([]string{pq.QuoteIdentifier(env.Name), "roles"}, ".")
	sqlStatement := fmt.Sprintf(`SELECT * FROM %s WHERE role_name=$1;`, tableName)
	var role pgRole
	row := pgds.Db.QueryRow(sqlStatement, roleName)
	err := row.Scan(&role.id, &role.Name, &role.description)
	switch err {
	// must define a generic error ("Not Found")
	case sql.ErrNoRows:
		fmt.Println("No rows were returned!")
		return rbac.Role{}, 0, err
	case nil:
		return role.Role, role.id, nil
	default:
		return rbac.Role{}, 0, err
	}
}

// TBD split into functions - this is too long
func (pgds *PostgresDataStore) CreateRole(env env.Environment, role rbac.Role) (int, error) {
	tableName := strings.Join([]string{pq.QuoteIdentifier(env.Name), "roles"}, ".")
	sqlStatement := fmt.Sprintf(`
	WITH inserted AS (
		INSERT INTO %s (role_name, description)
		VALUES ($1, $2)
		ON CONFLICT (role_name) DO NOTHING
		RETURNING id
	)
	SELECT id FROM inserted
	UNION ALL
	SELECT id FROM %s WHERE role_name = $1 AND NOT EXISTS (SELECT 1 FROM inserted);
	`, tableName, tableName)

	var roleId int
	err := pgds.Db.QueryRow(sqlStatement, role.Name, "placeholder description").Scan(&roleId)
	if err != nil {
		return 0, err
	}

	// create permissions
	perms := make([]string, len(role.Permissions))
	permQuotedNames := make([]string, len(role.Permissions))

	for i, perm := range role.Permissions {
		perms[i] = fmt.Sprintf("('%s', 'placeholder description')", perm)
		permQuotedNames[i] = fmt.Sprintf("'%s'", perm)
	}

	// TBD make safer by using $1, $2, $3, etc dynamically
	tableName = strings.Join([]string{pq.QuoteIdentifier(env.Name), "permissions"}, ".")
	sqlStatement = fmt.Sprintf(`
	WITH inserted AS (
		INSERT INTO  %s (permission_name, description)
		VALUES %s
		ON CONFLICT (permission_name) DO NOTHING
		RETURNING id
    )
	SELECT id FROM inserted
	UNION ALL 
	SELECT id FROM %s WHERE permission_name in (%s);
	`, tableName, strings.Join(perms, ","), tableName, strings.Join(permQuotedNames, ","))

	//log.Printf("\n \n SQL query to insert permissions : %v \n", sqlStatement)
	idRows, err := pgds.Db.Query(sqlStatement)
	if err != nil {
		log.Printf("error while inserting permissions : %v \n", err)
		return 0, err
	}

	permIds := []int{}
	for idRows.Next() {
		var permId int
		idRows.Scan(&permId)
		permIds = append(permIds, permId)
	}

	if err = idRows.Err(); err != nil {
		log.Printf("error while iterating over permissions : %v \n", err)
		return 0, err
	}

	// create permission-to-role records
	roleToPerms := make([]string, len(permIds))

	for i, permId := range permIds {
		roleToPerms[i] = fmt.Sprintf("(%v, %v )", roleId, permId)
	}

	tableName = strings.Join([]string{pq.QuoteIdentifier(env.Name), "role_permission"}, ".")
	sqlStatement = fmt.Sprintf(`
	INSERT INTO %s (role_id, permission_id)
	VALUES %s
	ON CONFLICT (role_id, permission_id) DO NOTHING
	`, tableName, strings.Join(roleToPerms, ","))

	_, err = pgds.Db.Query(sqlStatement)
	if err != nil {
		log.Printf("error while inserting role-to-permission mappings : %v \n", err)
		return 0, err
	}

	// clean-up deleted permission-role
	permIdsString := make([]string, len(permIds))
	for i, permId := range permIds {
		permIdsString[i] = strconv.Itoa(permId)
	}

	sqlStatement = fmt.Sprintf(`
	DELETE FROM %s
    WHERE role_id = %v
	AND permission_id NOT IN (%s)
	`, tableName, roleId, strings.Join(permIdsString, ","))

	_, err = pgds.Db.Exec(sqlStatement)
	if err != nil {
		log.Printf("error while deleting stale role-to-permission mappings : %v \n", err)
		return 0, err
	}

	// TBD - clean-up orphan permissions - maybe as a separate "cronjob" goroutine (unrelated to any specific operation)

	return roleId, nil

}

func (pgds *PostgresDataStore) DeleteRole(env env.Environment, roleName string) (int, error) {
	tableName := strings.Join([]string{pq.QuoteIdentifier(env.Name), "roles"}, ".")
	sqlStatement := fmt.Sprintf(`
	DELETE FROM %s
    WHERE role_name = $1 
	RETURNING id;	
	`, tableName)
	var id int
	err := pgds.Db.QueryRow(sqlStatement, roleName).Scan(&id)
	switch err {
	// must define a generic error ("Not Found")
	case sql.ErrNoRows:
		fmt.Println("Role to delete not found")
		return 0, nil
	case nil:
		return id, nil
	default:
		return 0, err
	}
}

// GRBinding
func (pgds *PostgresDataStore) GetGRBinding(env env.Environment, roleName string) (rbac.GroupRoleBinding, error) {
	return rbac.GroupRoleBinding{}, nil
}

func (pgds *PostgresDataStore) CreateGRBinding(env env.Environment, roleName string) (rbac.GroupRoleBinding, error) {
	return rbac.GroupRoleBinding{}, nil
}

func (pgds *PostgresDataStore) DeleteGRBinding(env env.Environment, roleName string) (rbac.GroupRoleBinding, error) {
	return rbac.GroupRoleBinding{}, nil
}
