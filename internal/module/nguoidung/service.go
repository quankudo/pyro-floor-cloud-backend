package nguoidung

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/pyrofloor/cloud-backend/internal/apperror"
	"github.com/pyrofloor/cloud-backend/internal/database"
	"github.com/pyrofloor/cloud-backend/internal/module/phanconggiamsat"
	"github.com/pyrofloor/cloud-backend/internal/module/tang"
	"github.com/pyrofloor/cloud-backend/internal/module/thanhvienkhuvuc"
)

const (
	MaxBatchUsers           = 25
	MaxUserNameRunes        = 150
	MaxPhoneRunes           = 15
	DefaultResidentPageSize = 20
	MaxResidentPageSize     = 100
)

type PasswordHasher func(password string) (string, error)

type AreaMembershipService interface {
	AddBatch(
		ctx context.Context,
		tx database.DBTX,
		input thanhvienkhuvuc.AddMembersInput,
	) error
}

type SupervisorAssignmentService interface {
	AddBatch(
		ctx context.Context,
		tx database.DBTX,
		input phanconggiamsat.AddAssignmentsInput,
	) error
}

type BuildingOwnerService interface {
	EnsureOwner(
		ctx context.Context,
		tx database.DBTX,
		buildingID, ownerID string,
	) error
}

type ResidentFloorLister interface {
	ListForBuilding(ctx context.Context, buildingID string) ([]tang.Tang, error)
}

const (
	DefaultSupervisorActorPageSize = 20
	MaxSupervisorActorPageSize     = 100
)

type Service struct {
	repository  BatchUserRepository
	residents   ResidentRepository
	actors      SupervisorActorRepository
	floors      ResidentFloorLister
	memberships AreaMembershipService
	assignments SupervisorAssignmentService
	buildings   BuildingOwnerService
	hash        PasswordHasher
}

type BatchUserRequest struct {
	TenNguoiDung string
	Email        string
	SoDienThoai  string
	MatKhau      string
	IDVaiTro     string
	KyHieuVaiTro string
	IDKhuVuc     string
	PhanCong     *ManagerAssignment
}

func NewService(
	repository BatchUserRepository,
	memberships AreaMembershipService,
	assignments SupervisorAssignmentService,
	buildings BuildingOwnerService,
	hash PasswordHasher,
	dependencies ...any,
) (*Service, error) {
	if repository == nil || memberships == nil || assignments == nil || buildings == nil || hash == nil {
		return nil, errors.New("batch user repository, relation services, building service and password hasher are required")
	}
	service := &Service{
		repository:  repository,
		memberships: memberships,
		assignments: assignments,
		buildings:   buildings,
		hash:        hash,
	}
	for _, dependency := range dependencies {
		if residentRepository, ok := dependency.(ResidentRepository); ok {
			service.residents = residentRepository
		}
		if supervisorActorRepository, ok := dependency.(SupervisorActorRepository); ok {
			service.actors = supervisorActorRepository
		}
		if floorLister, ok := dependency.(ResidentFloorLister); ok {
			service.floors = floorLister
		}
	}
	return service, nil
}

func (s *Service) CreateBatch(
	ctx context.Context,
	actorID string,
	actorRoleCode string,
	buildingID string,
	requests []BatchUserRequest,
) ([]CreatedUser, error) {
	actorRoleCode = strings.ToUpper(strings.TrimSpace(actorRoleCode))
	if !isUUID(actorID) {
		return nil, apperror.New(
			http.StatusUnauthorized,
			apperror.AppErrorUnauthenticated,
			"authentication is required",
			nil,
		)
	}
	if actorRoleCode != RoleCodeAdmin && actorRoleCode != RoleCodeOwner {
		return nil, apperror.New(
			http.StatusForbidden,
			apperror.AppErrorForbidden,
			"only an administrator or building owner can add batch members",
			nil,
		)
	}
	if len(requests) == 0 || len(requests) > MaxBatchUsers {
		return nil, ErrBatchSizeInvalid
	}
	buildingID = strings.TrimSpace(buildingID)
	if !isUUID(buildingID) {
		return nil, apperror.New(
			http.StatusBadRequest,
			apperror.AppErrorInvalidArgument,
			"id_toa_nha is required and must be a UUID",
			map[string]any{"field": "id_toa_nha"},
		)
	}
	users := make([]CreateUserInput, len(requests))
	seenEmails := make(map[string]struct{}, len(requests))
	roleCache := make(map[string]VaiTro, len(requests))
	for i, request := range requests {
		user, err := s.validateAndPrepareUser(ctx, actorRoleCode, request, roleCache)
		if err != nil {
			return nil, withBatchIndex(err, i)
		}
		if _, exists := seenEmails[user.Email]; exists {
			return nil, invalidUserField(i, "email", "email is duplicated in this batch")
		}
		seenEmails[user.Email] = struct{}{}
		user.PhanCong = copyManagerAssignment(user.PhanCong)
		users[i] = user
	}

	created := make([]CreatedUser, 0, len(users))
	err := s.repository.WithinTransaction(ctx, func(tx database.DBTX) error {
		created = make([]CreatedUser, 0, len(users))
		members := make([]thanhvienkhuvuc.AddMemberInput, 0, len(users))
		assignments := make([]phanconggiamsat.AddAssignmentInput, 0, len(users))
		if actorRoleCode == RoleCodeOwner {
			if err := s.buildings.EnsureOwner(ctx, tx, buildingID, actorID); err != nil {
				return err
			}
		}

		for i, user := range users {
			account, err := s.repository.FindOrCreateBatchUser(ctx, tx, user)
			if err != nil {
				return withBatchIndex(err, i)
			}
			if strings.ToUpper(strings.TrimSpace(account.KyHieuVaiTro)) != user.VaiTro.KyHieuVaiTro {
				return withBatchIndex(ErrExistingUserRoleMismatch, i)
			}

			switch user.VaiTro.KyHieuVaiTro {
			case RoleCodeResident:
				members = append(members, thanhvienkhuvuc.AddMemberInput{
					UserID: account.IDNguoiDung,
					AreaID: user.IDKhuVuc,
				})
			case RoleCodeManager:
				assignments = append(assignments, phanconggiamsat.AddAssignmentInput{
					UserID:  account.IDNguoiDung,
					FloorID: user.PhanCong.IDTang,
					AreaID:  user.PhanCong.IDKhuVuc,
				})
			default:
				return withBatchIndex(ErrRoleNotAllowed, i)
			}
			created = append(created, account)
		}

		if len(members) > 0 {
			if err := s.memberships.AddBatch(ctx, tx, thanhvienkhuvuc.AddMembersInput{
				BuildingID: buildingID,
				Members:    members,
			}); err != nil {
				return err
			}
		}
		if len(assignments) > 0 {
			if err := s.assignments.AddBatch(ctx, tx, phanconggiamsat.AddAssignmentsInput{
				BuildingID:  buildingID,
				Assignments: assignments,
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func copyManagerAssignment(input *ManagerAssignment) *ManagerAssignment {
	if input == nil {
		return nil
	}
	copied := *input
	return &copied
}

func (s *Service) ListResidents(
	ctx context.Context,
	buildingID string,
	filter ResidentFilter,
) (*ResidentPage, error) {
	buildingID = strings.TrimSpace(buildingID)
	if !isUUID(buildingID) {
		return nil, apperror.New(
			http.StatusForbidden,
			apperror.AppErrorForbidden,
			"building scope is missing from authentication token",
			nil,
		)
	}
	if s.residents == nil {
		return nil, errors.New("resident repository is not configured")
	}
	if filter.Limit == 0 {
		filter.Limit = DefaultResidentPageSize
	}
	if filter.Limit < 1 || filter.Limit > MaxResidentPageSize {
		return nil, apperror.New(
			http.StatusBadRequest,
			apperror.AppErrorInvalidArgument,
			"limit must be between 1 and 100",
			map[string]any{"field": "limit"},
		)
	}
	if filter.Offset < 0 {
		return nil, apperror.New(
			http.StatusBadRequest,
			apperror.AppErrorInvalidArgument,
			"offset must not be negative",
			map[string]any{"field": "offset"},
		)
	}

	if filter.FloorID != nil {
		value := strings.TrimSpace(*filter.FloorID)
		if value == "" {
			filter.FloorID = nil
		} else {
			if !isUUID(value) {
				return nil, apperror.New(
					http.StatusBadRequest,
					apperror.AppErrorInvalidArgument,
					"floor ID must be a valid UUID",
					map[string]any{"field": "id_tang"},
				)
			}
			filter.FloorID = &value
		}
	}
	if filter.AreaID != nil {
		value := strings.TrimSpace(*filter.AreaID)
		if value == "" {
			filter.AreaID = nil
		} else {
			if !isUUID(value) {
				return nil, apperror.New(
					http.StatusBadRequest,
					apperror.AppErrorInvalidArgument,
					"area ID must be a valid UUID",
					map[string]any{"field": "id_khu_vuc"},
				)
			}
			filter.AreaID = &value
		}
	}
	normalizeResidentTextFilter(&filter.Status)
	normalizeResidentTextFilter(&filter.SearchKeyword)

	items, total, err := s.residents.ListResidents(ctx, buildingID, filter)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []Resident{}
	}
	page := &ResidentPage{
		Items:  items,
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
		Floors: []ResidentFloor{},
	}
	if filter.IncludeFloors {
		if s.floors == nil {
			return nil, errors.New("floor lister is not configured")
		}
		floors, err := s.floors.ListForBuilding(ctx, buildingID)
		if err != nil {
			return nil, err
		}
		page.Floors = make([]ResidentFloor, len(floors))
		for i, floor := range floors {
			page.Floors[i] = ResidentFloor{
				IDTang: floor.IDTang,
				SoTang: floor.SoTang,
			}
		}
	}
	return page, nil
}

func (s *Service) ListSupervisorActors(
	ctx context.Context,
	buildingID string,
	filter SupervisorActorFilter,
) (*SupervisorActorPage, error) {
	buildingID = strings.TrimSpace(buildingID)
	if !isUUID(buildingID) {
		return nil, apperror.New(
			http.StatusForbidden,
			apperror.AppErrorForbidden,
			"building scope is missing from authentication token",
			nil,
		)
	}
	if s.actors == nil {
		return nil, errors.New("supervisor actor repository is not configured")
	}
	if filter.Limit == 0 {
		filter.Limit = DefaultSupervisorActorPageSize
	}
	if filter.Limit < 1 || filter.Limit > MaxSupervisorActorPageSize {
		return nil, apperror.New(
			http.StatusBadRequest,
			apperror.AppErrorInvalidArgument,
			"limit must be between 1 and 100",
			map[string]any{"field": "limit"},
		)
	}
	if filter.Offset < 0 {
		return nil, apperror.New(
			http.StatusBadRequest,
			apperror.AppErrorInvalidArgument,
			"offset must not be negative",
			map[string]any{"field": "offset"},
		)
	}
	normalizeSupervisorActorTextFilter(&filter.SearchKeyword)
	normalizeSupervisorActorTextFilter(&filter.Status)

	items, total, err := s.actors.ListSupervisorActors(ctx, buildingID, filter)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []SupervisorActor{}
	}
	return &SupervisorActorPage{
		Items:  items,
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}, nil
}

func normalizeSupervisorActorTextFilter(value **string) {
	if *value == nil {
		return
	}
	trimmed := strings.TrimSpace(**value)
	if trimmed == "" {
		*value = nil
		return
	}
	*value = &trimmed
}

func normalizeResidentTextFilter(value **string) {
	if *value == nil {
		return
	}
	trimmed := strings.TrimSpace(**value)
	if trimmed == "" {
		*value = nil
		return
	}
	*value = &trimmed
}

func (s *Service) validateAndPrepareUser(
	ctx context.Context,
	actorRoleCode string,
	request BatchUserRequest,
	roleCache map[string]VaiTro,
) (CreateUserInput, error) {
	name := strings.TrimSpace(request.TenNguoiDung)
	if name == "" || !utf8.ValidString(name) ||
		utf8.RuneCountInString(name) > MaxUserNameRunes {
		return CreateUserInput{}, apperror.New(
			http.StatusBadRequest,
			apperror.AppErrorInvalidArgument,
			"name is required and must be at most 150 characters",
			map[string]any{"field": "ten_nguoi_dung"},
		)
	}

	email, err := normalizeUserEmail(request.Email)
	if err != nil {
		return CreateUserInput{}, apperror.New(
			http.StatusBadRequest,
			apperror.AppErrorInvalidArgument,
			"email must be valid",
			map[string]any{"field": "email"},
		)
	}

	if request.MatKhau == "" || !utf8.ValidString(request.MatKhau) ||
		len(request.MatKhau) < 8 || len(request.MatKhau) > 72 {
		return CreateUserInput{}, apperror.New(
			http.StatusBadRequest,
			apperror.AppErrorInvalidArgument,
			"password must be between 8 and 72 bytes",
			map[string]any{"field": "mat_khau"},
		)
	}
	phone := strings.TrimSpace(request.SoDienThoai)
	if !utf8.ValidString(phone) || utf8.RuneCountInString(phone) > MaxPhoneRunes {
		return CreateUserInput{}, apperror.New(
			http.StatusBadRequest,
			apperror.AppErrorInvalidArgument,
			"phone number must be at most 15 characters",
			map[string]any{"field": "so_dien_thoai"},
		)
	}

	roleID := strings.TrimSpace(request.IDVaiTro)
	roleCode := strings.ToUpper(strings.TrimSpace(request.KyHieuVaiTro))
	if (roleID == "") == (roleCode == "") {
		return CreateUserInput{}, apperror.New(
			http.StatusBadRequest,
			apperror.AppErrorInvalidArgument,
			"exactly one of id_vai_tro or ky_hieu_vai_tro is required",
			map[string]any{"fields": []string{"id_vai_tro", "ky_hieu_vai_tro"}},
		)
	}
	if roleID != "" && !isUUID(roleID) {
		return CreateUserInput{}, apperror.New(
			http.StatusBadRequest,
			apperror.AppErrorInvalidArgument,
			"id_vai_tro must be a valid UUID",
			map[string]any{"field": "id_vai_tro"},
		)
	}
	if roleCode != "" && len(roleCode) > 30 {
		return CreateUserInput{}, apperror.New(
			http.StatusBadRequest,
			apperror.AppErrorInvalidArgument,
			"ky_hieu_vai_tro must be at most 30 characters",
			map[string]any{"field": "ky_hieu_vai_tro"},
		)
	}

	cacheKey := "id:" + roleID
	if roleCode != "" {
		cacheKey = "code:" + roleCode
	}
	role, ok := roleCache[cacheKey]
	if !ok {
		role, err = s.repository.ResolveRole(ctx, roleID, roleCode)
		if err != nil {
			if apperror.IsCode(err, apperror.AppErrorNotFound) {
				return CreateUserInput{}, ErrRoleNotFound
			}
			return CreateUserInput{}, fmt.Errorf("resolve role: %w", err)
		}
		role.KyHieuVaiTro = strings.ToUpper(strings.TrimSpace(role.KyHieuVaiTro))
		roleCache[cacheKey] = role
	}
	if actorRoleCode == RoleCodeOwner &&
		role.KyHieuVaiTro != RoleCodeManager &&
		role.KyHieuVaiTro != RoleCodeResident {
		return CreateUserInput{}, apperror.New(
			http.StatusForbidden,
			apperror.AppErrorForbidden,
			"owners may only create QUAN_LY or CU_DAN users",
			map[string]any{"field": "ky_hieu_vai_tro"},
		)
	}
	if role.KyHieuVaiTro != RoleCodeManager && role.KyHieuVaiTro != RoleCodeResident {
		return CreateUserInput{}, ErrRoleNotAllowed
	}

	areaID := strings.TrimSpace(request.IDKhuVuc)
	if areaID != "" && !isUUID(areaID) {
		return CreateUserInput{}, apperror.New(
			http.StatusBadRequest,
			apperror.AppErrorInvalidArgument,
			"id_khu_vuc must be a valid UUID",
			map[string]any{"field": "id_khu_vuc"},
		)
	}
	switch role.KyHieuVaiTro {
	case RoleCodeResident:
		if areaID == "" {
			return CreateUserInput{}, apperror.New(
				http.StatusBadRequest,
				apperror.AppErrorInvalidArgument,
				"residents must include id_khu_vuc",
				map[string]any{"field": "id_khu_vuc"},
			)
		}
		if request.PhanCong != nil {
			return CreateUserInput{}, apperror.New(
				http.StatusBadRequest,
				apperror.AppErrorInvalidArgument,
				"residents must use id_khu_vuc instead of phan_cong",
				nil,
			)
		}
	case RoleCodeManager:
		if areaID != "" {
			return CreateUserInput{}, apperror.New(
				http.StatusBadRequest,
				apperror.AppErrorInvalidArgument,
				"managers must use phan_cong instead of id_khu_vuc",
				nil,
			)
		}
		if request.PhanCong == nil {
			return CreateUserInput{}, apperror.New(
				http.StatusBadRequest,
				apperror.AppErrorInvalidArgument,
				"managers must include phan_cong",
				map[string]any{"field": "phan_cong"},
			)
		}
		if request.PhanCong != nil {
			if request.PhanCong.IDTang != "" &&
				!isUUID(strings.TrimSpace(request.PhanCong.IDTang)) {
				return CreateUserInput{}, invalidAssignmentField("id_tang")
			}
			if request.PhanCong.IDKhuVuc != "" &&
				!isUUID(strings.TrimSpace(request.PhanCong.IDKhuVuc)) {
				return CreateUserInput{}, invalidAssignmentField("id_khu_vuc")
			}
			request.PhanCong.IDTang = strings.TrimSpace(request.PhanCong.IDTang)
			request.PhanCong.IDKhuVuc = strings.TrimSpace(request.PhanCong.IDKhuVuc)
		}
	default:
		if areaID != "" || request.PhanCong != nil {
			return CreateUserInput{}, apperror.New(
				http.StatusBadRequest,
				apperror.AppErrorInvalidArgument,
				"assignment fields are only supported for QUAN_LY or CU_DAN",
				nil,
			)
		}
	}

	passwordHash, err := s.hash(request.MatKhau)
	if err != nil {
		return CreateUserInput{}, fmt.Errorf("hash user password: %w", err)
	}

	return CreateUserInput{
		TenNguoiDung: name,
		Email:        email,
		SoDienThoai:  phone,
		MatKhauHash:  passwordHash,
		VaiTro:       role,
		IDKhuVuc:     areaID,
		PhanCong:     request.PhanCong,
	}, nil
}

func normalizeUserEmail(value string) (string, error) {
	value = strings.TrimSpace(value)
	if len(value) > 255 {
		return "", errors.New("email is too long")
	}
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address != value || !strings.Contains(value, "@") {
		return "", errors.New("invalid email")
	}
	return strings.ToLower(value), nil
}

func withBatchIndex(err error, index int) error {
	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr == nil {
		return fmt.Errorf("users[%d]: %w", index, err)
	}
	details := make(map[string]any, len(appErr.Detail)+1)
	for key, value := range appErr.Detail {
		details[key] = value
	}
	details["index"] = index
	return apperror.Wrap(
		appErr.HTTPStatus,
		appErr.Code,
		appErr.Message,
		details,
		appErr.Err,
	)
}

func invalidUserField(index int, field, message string) error {
	return apperror.New(
		http.StatusBadRequest,
		apperror.AppErrorInvalidArgument,
		message,
		map[string]any{"index": index, "field": field},
	)
}

func invalidAssignmentField(field string) error {
	return apperror.New(
		http.StatusBadRequest,
		apperror.AppErrorInvalidArgument,
		"assignment field must be a valid UUID",
		map[string]any{"field": "phan_cong." + field},
	)
}

func isUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for i, char := range value {
		switch i {
		case 8, 13, 18, 23:
			if char != '-' {
				return false
			}
		default:
			if !((char >= '0' && char <= '9') ||
				(char >= 'a' && char <= 'f') ||
				(char >= 'A' && char <= 'F')) {
				return false
			}
		}
	}
	return true
}
