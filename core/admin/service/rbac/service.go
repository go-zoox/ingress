package rbac

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-zoox/gormx"
	"github.com/go-zoox/ingress/core/admin/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	defaultAdminUsername = "admin"
	defaultAdminPassword = "admin"
)

// SeedOptions configures the auth.basic bootstrap user synced on every startup.
type SeedOptions struct {
	BasicUsername string
	BasicPassword string
}

// PermissionRow is a list entry for permission management.
type PermissionRow struct {
	ID          uint   `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Group       string `json:"group"`
	Description string `json:"description,omitempty"`
	Builtin     bool   `json:"builtin"`
}

// RoleRow is a list entry for role management.
type RoleRow struct {
	ID            uint     `json:"id"`
	Code          string   `json:"code"`
	Name          string   `json:"name"`
	Description   string   `json:"description,omitempty"`
	Builtin       bool     `json:"builtin"`
	PermissionIDs []uint   `json:"permission_ids"`
	Permissions   []string `json:"permissions,omitempty"`
	UserCount     int      `json:"user_count"`
}

// UserRow is a list entry for user management.
type UserRow struct {
	ID          uint     `json:"id"`
	Username    string   `json:"username"`
	DisplayName string   `json:"display_name"`
	Email       string   `json:"email,omitempty"`
	Enabled     bool     `json:"enabled"`
	Builtin     bool     `json:"builtin"`
	RoleIDs     []uint   `json:"role_ids"`
	Roles       []string `json:"roles,omitempty"`
}

// PermissionInput creates or updates a custom permission.
type PermissionInput struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Group       string `json:"group"`
	Description string `json:"description"`
}

// RoleInput creates or updates a role.
type RoleInput struct {
	Code          string `json:"code"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	PermissionIDs []uint `json:"permission_ids"`
}

// UserInput creates or updates a user.
type UserInput struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	Enabled     bool   `json:"enabled"`
	RoleIDs     []uint `json:"role_ids"`
}

// PasswordInput resets a user password.
type PasswordInput struct {
	Password string `json:"password"`
}

// Service manages RBAC entities in SQLite.
type Service struct{}

func New() *Service {
	return &Service{}
}

func db() *gorm.DB {
	return gormx.GetDB()
}

// Seed ensures builtin permissions, roles, and the default admin user exist.
func (s *Service) Seed(opts SeedOptions) error {
	if err := s.syncBuiltinPermissions(); err != nil {
		return err
	}
	if err := s.ensureBuiltinRoles(); err != nil {
		return err
	}
	return s.ensureDefaultAdminUser(opts)
}

func (s *Service) syncBuiltinPermissions() error {
	for _, def := range AllBuiltinPermissions() {
		var row model.RBACPermission
		err := db().Where("code = ?", def.Code).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			row = model.RBACPermission{
				Code:        def.Code,
				Name:        def.Name,
				Group:       def.Group,
				Description: def.Description,
				Builtin:     true,
			}
			if err := db().Create(&row).Error; err != nil {
				return fmt.Errorf("rbac: seed permission %s: %w", def.Code, err)
			}
			continue
		}
		if err != nil {
			return err
		}
		updates := map[string]any{
			"name":        def.Name,
			"group":       def.Group,
			"description": def.Description,
			"builtin":     true,
		}
		if err := db().Model(&row).Updates(updates).Error; err != nil {
			return fmt.Errorf("rbac: sync permission %s: %w", def.Code, err)
		}
	}
	return nil
}

func (s *Service) ensureBuiltinRoles() error {
	allPerms, err := s.loadAllPermissions()
	if err != nil {
		return err
	}
	for _, def := range builtinRoleDefs() {
		perms, err := resolveRolePermissions(allPerms, def.Permissions)
		if err != nil {
			return err
		}
		if err := s.ensureRole(def.Code, def.Name, def.Description, true, perms); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) ensureRole(code, name, desc string, builtin bool, perms []model.RBACPermission) error {
	var role model.RBACRole
	err := db().Where("code = ?", code).First(&role).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		role = model.RBACRole{
			Code:        code,
			Name:        name,
			Description: desc,
			Builtin:     builtin,
		}
		if err := db().Create(&role).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		updates := map[string]any{
			"name":        name,
			"description": desc,
			"builtin":     builtin,
		}
		if err := db().Model(&role).Updates(updates).Error; err != nil {
			return err
		}
	}
	return db().Model(&role).Association("Permissions").Replace(perms)
}

func (s *Service) ensureDefaultAdminUser(opts SeedOptions) error {
	username := defaultAdminUsername
	password := defaultAdminPassword
	if strings.TrimSpace(opts.BasicUsername) != "" {
		username = normalizeUsername(opts.BasicUsername)
	}
	if strings.TrimSpace(opts.BasicPassword) != "" {
		password = opts.BasicPassword
	}

	var adminRole model.RBACRole
	if err := db().Where("code = ?", adminRoleCode).First(&adminRole).Error; err != nil {
		return err
	}

	var user model.RBACUser
	err := db().Preload("Roles").Where("username = ?", username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		hash, err := hashPassword(password)
		if err != nil {
			return err
		}
		user = model.RBACUser{
			Username:     username,
			DisplayName:  "系统管理员",
			PasswordHash: hash,
			Enabled:      true,
			Builtin:      true,
		}
		if err := db().Create(&user).Error; err != nil {
			return err
		}
		return db().Model(&user).Association("Roles").Replace([]model.RBACRole{adminRole})
	}
	if err != nil {
		return err
	}

	updates := map[string]any{}
	if !user.Builtin {
		updates["builtin"] = true
	}
	if len(updates) > 0 {
		if err := db().Model(&user).Updates(updates).Error; err != nil {
			return err
		}
	}

	hasAdminRole := false
	roles := append([]model.RBACRole(nil), user.Roles...)
	for _, role := range roles {
		if role.Code == adminRoleCode {
			hasAdminRole = true
			break
		}
	}
	if !hasAdminRole {
		roles = append(roles, adminRole)
		if err := db().Model(&user).Association("Roles").Replace(roles); err != nil {
			return err
		}
	}
	return nil
}

// Authenticate validates RBAC credentials for basic auth login.
func (s *Service) Authenticate(username, password string) (*UserRow, error) {
	username = normalizeUsername(username)
	password = strings.TrimSpace(password)
	if username == "" || password == "" {
		return nil, fmt.Errorf("invalid credentials")
	}
	var user model.RBACUser
	err := db().Preload("Roles").Where("username = ? AND enabled = ?", username, true).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("invalid credentials")
	}
	if err != nil {
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}
	row := userToRow(user)
	return &row, nil
}

func (s *Service) ListPermissions() ([]PermissionRow, error) {
	var rows []model.RBACPermission
	if err := db().Order("`group` asc, code asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]PermissionRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, PermissionRow{
			ID:          row.ID,
			Code:        row.Code,
			Name:        row.Name,
			Group:       row.Group,
			Description: row.Description,
			Builtin:     row.Builtin,
		})
	}
	return out, nil
}

func (s *Service) CreatePermission(in PermissionInput) (*PermissionRow, error) {
	code := normalizeCode(in.Code)
	if code == "" {
		return nil, fmt.Errorf("permission code is required")
	}
	if in.Name = strings.TrimSpace(in.Name); in.Name == "" {
		return nil, fmt.Errorf("permission name is required")
	}
	group := strings.TrimSpace(in.Group)
	if group == "" {
		group = "自定义"
	}
	row := model.RBACPermission{
		Code:        code,
		Name:        in.Name,
		Group:       group,
		Description: strings.TrimSpace(in.Description),
		Builtin:     false,
	}
	if err := db().Create(&row).Error; err != nil {
		return nil, err
	}
	return &PermissionRow{
		ID:          row.ID,
		Code:        row.Code,
		Name:        row.Name,
		Group:       row.Group,
		Description: row.Description,
		Builtin:     row.Builtin,
	}, nil
}

func (s *Service) UpdatePermission(id uint, in PermissionInput) (*PermissionRow, error) {
	row, err := s.getPermission(id)
	if err != nil {
		return nil, err
	}
	if row.Builtin {
		return nil, fmt.Errorf("builtin permission cannot be edited")
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, fmt.Errorf("permission name is required")
	}
	group := strings.TrimSpace(in.Group)
	if group == "" {
		group = row.Group
	}
	updates := map[string]any{
		"name":        name,
		"group":       group,
		"description": strings.TrimSpace(in.Description),
	}
	if err := db().Model(row).Updates(updates).Error; err != nil {
		return nil, err
	}
	return &PermissionRow{
		ID:          row.ID,
		Code:        row.Code,
		Name:        name,
		Group:       group,
		Description: strings.TrimSpace(in.Description),
		Builtin:     row.Builtin,
	}, nil
}

func (s *Service) DeletePermission(id uint) error {
	row, err := s.getPermission(id)
	if err != nil {
		return err
	}
	if row.Builtin {
		return fmt.Errorf("builtin permission cannot be deleted")
	}
	count, err := s.permissionRoleCount(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("permission is assigned to %d role(s)", count)
	}
	return db().Delete(row).Error
}

func (s *Service) ListRoles() ([]RoleRow, error) {
	var rows []model.RBACRole
	if err := db().Preload("Permissions").Order("code asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]RoleRow, 0, len(rows))
	for _, row := range rows {
		userCount, err := s.roleUserCount(row.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, roleToRow(row, userCount))
	}
	return out, nil
}

func (s *Service) CreateRole(in RoleInput) (*RoleRow, error) {
	code := normalizeCode(in.Code)
	if code == "" {
		return nil, fmt.Errorf("role code is required")
	}
	if in.Name = strings.TrimSpace(in.Name); in.Name == "" {
		return nil, fmt.Errorf("role name is required")
	}
	perms, err := s.permissionsByIDs(in.PermissionIDs)
	if err != nil {
		return nil, err
	}
	row := model.RBACRole{
		Code:        code,
		Name:        in.Name,
		Description: strings.TrimSpace(in.Description),
		Builtin:     false,
	}
	if err := db().Create(&row).Error; err != nil {
		return nil, err
	}
	if err := db().Model(&row).Association("Permissions").Replace(perms); err != nil {
		return nil, err
	}
	row.Permissions = perms
	return ptr(roleToRow(row, 0)), nil
}

func (s *Service) UpdateRole(id uint, in RoleInput) (*RoleRow, error) {
	row, err := s.getRole(id)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, fmt.Errorf("role name is required")
	}
	perms, err := s.permissionsByIDs(in.PermissionIDs)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{
		"name":        name,
		"description": strings.TrimSpace(in.Description),
	}
	if !row.Builtin {
		code := normalizeCode(in.Code)
		if code == "" {
			return nil, fmt.Errorf("role code is required")
		}
		updates["code"] = code
	}
	if err := db().Model(row).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := db().Model(row).Association("Permissions").Replace(perms); err != nil {
		return nil, err
	}
	if err := db().Preload("Permissions").First(row, id).Error; err != nil {
		return nil, err
	}
	userCount, _ := s.roleUserCount(row.ID)
	return ptr(roleToRow(*row, userCount)), nil
}

func (s *Service) DeleteRole(id uint) error {
	row, err := s.getRole(id)
	if err != nil {
		return err
	}
	if row.Builtin {
		return fmt.Errorf("builtin role cannot be deleted")
	}
	count, err := s.roleUserCount(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("role is assigned to %d user(s)", count)
	}
	return db().Select("Permissions").Delete(row).Error
}

func (s *Service) ListUsers() ([]UserRow, error) {
	var rows []model.RBACUser
	if err := db().Preload("Roles").Order("username asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]UserRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, userToRow(row))
	}
	return out, nil
}

func (s *Service) CreateUser(in UserInput) (*UserRow, error) {
	username := normalizeUsername(in.Username)
	if username == "" {
		return nil, fmt.Errorf("username is required")
	}
	displayName := strings.TrimSpace(in.DisplayName)
	if displayName == "" {
		displayName = username
	}
	password := strings.TrimSpace(in.Password)
	if password == "" {
		return nil, fmt.Errorf("password is required")
	}
	if len(password) < 6 {
		return nil, fmt.Errorf("password must be at least 6 characters")
	}
	hash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}
	roles, err := s.rolesByIDs(in.RoleIDs)
	if err != nil {
		return nil, err
	}
	row := model.RBACUser{
		Username:     username,
		DisplayName:  displayName,
		Email:        strings.TrimSpace(in.Email),
		PasswordHash: hash,
		Enabled:      in.Enabled,
		Builtin:      false,
	}
	if err := db().Create(&row).Error; err != nil {
		return nil, err
	}
	if err := db().Model(&row).Association("Roles").Replace(roles); err != nil {
		return nil, err
	}
	row.Roles = roles
	return ptr(userToRow(row)), nil
}

func (s *Service) UpdateUser(id uint, in UserInput) (*UserRow, error) {
	row, err := s.getUser(id)
	if err != nil {
		return nil, err
	}
	displayName := strings.TrimSpace(in.DisplayName)
	if displayName == "" {
		return nil, fmt.Errorf("display name is required")
	}
	roles, err := s.rolesByIDs(in.RoleIDs)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{
		"display_name": displayName,
		"email":        strings.TrimSpace(in.Email),
		"enabled":      in.Enabled,
	}
	if !row.Builtin {
		username := normalizeUsername(in.Username)
		if username == "" {
			return nil, fmt.Errorf("username is required")
		}
		updates["username"] = username
	}
	if err := db().Model(row).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := db().Model(row).Association("Roles").Replace(roles); err != nil {
		return nil, err
	}
	if err := db().Preload("Roles").First(row, id).Error; err != nil {
		return nil, err
	}
	return ptr(userToRow(*row)), nil
}

func (s *Service) UpdateUserPassword(id uint, in PasswordInput) error {
	row, err := s.getUser(id)
	if err != nil {
		return err
	}
	password := strings.TrimSpace(in.Password)
	if password == "" {
		return fmt.Errorf("password is required")
	}
	if len(password) < 6 {
		return fmt.Errorf("password must be at least 6 characters")
	}
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	return db().Model(row).Update("password_hash", hash).Error
}

func (s *Service) DeleteUser(id uint) error {
	row, err := s.getUser(id)
	if err != nil {
		return err
	}
	if row.Builtin {
		return fmt.Errorf("builtin user cannot be deleted")
	}
	return db().Select("Roles").Delete(row).Error
}

func (s *Service) getPermission(id uint) (*model.RBACPermission, error) {
	var row model.RBACPermission
	if err := db().First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("permission not found")
		}
		return nil, err
	}
	return &row, nil
}

func (s *Service) getRole(id uint) (*model.RBACRole, error) {
	var row model.RBACRole
	if err := db().First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("role not found")
		}
		return nil, err
	}
	return &row, nil
}

func (s *Service) getUser(id uint) (*model.RBACUser, error) {
	var row model.RBACUser
	if err := db().First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return &row, nil
}

func (s *Service) loadAllPermissions() ([]model.RBACPermission, error) {
	var rows []model.RBACPermission
	if err := db().Order("code asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Service) permissionsByIDs(ids []uint) ([]model.RBACPermission, error) {
	if len(ids) == 0 {
		return []model.RBACPermission{}, nil
	}
	var rows []model.RBACPermission
	if err := db().Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) != len(uniqueUints(ids)) {
		return nil, fmt.Errorf("one or more permissions not found")
	}
	return rows, nil
}

func (s *Service) rolesByIDs(ids []uint) ([]model.RBACRole, error) {
	if len(ids) == 0 {
		return []model.RBACRole{}, nil
	}
	var rows []model.RBACRole
	if err := db().Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) != len(uniqueUints(ids)) {
		return nil, fmt.Errorf("one or more roles not found")
	}
	return rows, nil
}

func (s *Service) permissionRoleCount(id uint) (int, error) {
	var n int64
	err := db().Table("rbac_role_permissions").Where("rbac_permission_id = ?", id).Count(&n).Error
	return int(n), err
}

func (s *Service) roleUserCount(id uint) (int, error) {
	var n int64
	err := db().Table("rbac_user_roles").Where("rbac_role_id = ?", id).Count(&n).Error
	return int(n), err
}

func roleToRow(row model.RBACRole, userCount int) RoleRow {
	permIDs := make([]uint, 0, len(row.Permissions))
	codes := make([]string, 0, len(row.Permissions))
	for _, p := range row.Permissions {
		permIDs = append(permIDs, p.ID)
		codes = append(codes, p.Code)
	}
	return RoleRow{
		ID:            row.ID,
		Code:          row.Code,
		Name:          row.Name,
		Description:   row.Description,
		Builtin:       row.Builtin,
		PermissionIDs: permIDs,
		Permissions:   codes,
		UserCount:     userCount,
	}
}

func userToRow(row model.RBACUser) UserRow {
	roleIDs := make([]uint, 0, len(row.Roles))
	names := make([]string, 0, len(row.Roles))
	for _, r := range row.Roles {
		roleIDs = append(roleIDs, r.ID)
		names = append(names, r.Name)
	}
	return UserRow{
		ID:          row.ID,
		Username:    row.Username,
		DisplayName: row.DisplayName,
		Email:       row.Email,
		Enabled:     row.Enabled,
		Builtin:     row.Builtin,
		RoleIDs:     roleIDs,
		Roles:       names,
	}
}

func hashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func normalizeCode(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func normalizeUsername(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func uniqueUints(ids []uint) []uint {
	seen := make(map[uint]struct{}, len(ids))
	out := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func ptr[T any](v T) *T {
	return &v
}
