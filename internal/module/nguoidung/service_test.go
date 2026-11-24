package nguoidung

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/pyrofloor/cloud-backend/internal/apperror"
	"github.com/pyrofloor/cloud-backend/internal/database"
	"github.com/pyrofloor/cloud-backend/internal/module/phanconggiamsat"
	"github.com/pyrofloor/cloud-backend/internal/module/thanhvienkhuvuc"
)

const (
	testActorID    = "550e8400-e29b-41d4-a716-446655440002"
	testBuildingID = "550e8400-e29b-41d4-a716-446655440030"
	testAreaID     = "550e8400-e29b-41d4-a716-446655440020"
	testFloorID    = "550e8400-e29b-41d4-a716-446655440031"
)

type fakeBatchUserRepository struct {
	roles          map[string]VaiTro
	existing       map[string]CreatedUser
	created        []CreateUserInput
	transaction    int
	committed      int
	rolledBack     int
	findOrCreate   int
	transactionErr error
}

type fakeBuildingOwnerService struct {
	calls    int
	building string
	owner    string
	err      error
}

func (s *fakeBuildingOwnerService) EnsureOwner(
	_ context.Context,
	_ database.DBTX,
	buildingID, ownerID string,
) error {
	s.calls++
	s.building = buildingID
	s.owner = ownerID
	return s.err
}

func (r *fakeBatchUserRepository) ResolveRole(
	_ context.Context,
	roleID, roleCode string,
) (VaiTro, error) {
	key := "code:" + roleCode
	if roleID != "" {
		key = "id:" + roleID
	}
	role, ok := r.roles[key]
	if !ok {
		return VaiTro{}, ErrRoleNotFound
	}
	return role, nil
}

func (r *fakeBatchUserRepository) WithinTransaction(
	_ context.Context,
	fn func(database.DBTX) error,
) error {
	r.transaction++
	if err := fn(nil); err != nil {
		r.rolledBack++
		return err
	}
	if r.transactionErr != nil {
		r.rolledBack++
		return r.transactionErr
	}
	r.committed++
	return nil
}

func (r *fakeBatchUserRepository) FindOrCreateBatchUser(
	_ context.Context,
	_ database.DBTX,
	input CreateUserInput,
) (CreatedUser, error) {
	r.findOrCreate++
	if existing, ok := r.existing[input.Email]; ok {
		return existing, nil
	}
	r.created = append(r.created, input)
	return CreatedUser{
		NguoiDung: NguoiDung{
			IDNguoiDung:  "550e8400-e29b-41d4-a716-446655440010",
			IDVaiTro:     input.VaiTro.IDVaiTro,
			TenNguoiDung: input.TenNguoiDung,
			Email:        input.Email,
			TrangThai:    HoatDong,
		},
		KyHieuVaiTro: input.VaiTro.KyHieuVaiTro,
	}, nil
}

type fakeMembershipService struct {
	calls  int
	inputs []thanhvienkhuvuc.AddMembersInput
	err    error
}

func (s *fakeMembershipService) AddBatch(
	_ context.Context,
	_ database.DBTX,
	input thanhvienkhuvuc.AddMembersInput,
) error {
	s.calls++
	s.inputs = append(s.inputs, input)
	return s.err
}

type fakeAssignmentService struct {
	calls  int
	inputs []phanconggiamsat.AddAssignmentsInput
	err    error
}

func (s *fakeAssignmentService) AddBatch(
	_ context.Context,
	_ database.DBTX,
	input phanconggiamsat.AddAssignmentsInput,
) error {
	s.calls++
	s.inputs = append(s.inputs, input)
	return s.err
}

func newTestService(
	t *testing.T,
	repository *fakeBatchUserRepository,
	memberships *fakeMembershipService,
	assignments *fakeAssignmentService,
) *Service {
	t.Helper()
	service, err := NewService(
		repository,
		memberships,
		assignments,
		&fakeBuildingOwnerService{},
		func(password string) (string, error) {
			return "hashed:" + password, nil
		},
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}

func validBatchRequest(roleCode, email string) BatchUserRequest {
	request := BatchUserRequest{
		TenNguoiDung: "User One",
		Email:        email,
		SoDienThoai:  "0123456789",
		MatKhau:      "password-123",
		KyHieuVaiTro: roleCode,
	}
	switch roleCode {
	case RoleCodeResident:
		request.IDKhuVuc = testAreaID
	case RoleCodeManager:
		request.PhanCong = &ManagerAssignment{}
	}
	return request
}

func TestServiceBatchesMembershipsAndAssignmentsInOneTransaction(t *testing.T) {
	repository := &fakeBatchUserRepository{
		roles: map[string]VaiTro{
			"code:CU_DAN": {
				IDVaiTro:     "550e8400-e29b-41d4-a716-446655440001",
				KyHieuVaiTro: RoleCodeResident,
			},
			"code:QUAN_LY": {
				IDVaiTro:     "550e8400-e29b-41d4-a716-446655440004",
				KyHieuVaiTro: RoleCodeManager,
			},
		},
	}
	memberships := &fakeMembershipService{}
	assignments := &fakeAssignmentService{}
	buildingOwner := &fakeBuildingOwnerService{}
	service, err := NewService(
		repository,
		memberships,
		assignments,
		buildingOwner,
		func(password string) (string, error) { return "hashed:" + password, nil },
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	managerRequest := validBatchRequest(RoleCodeManager, "manager@example.com")
	managerRequest.PhanCong.IDTang = testFloorID
	managerRequest.PhanCong.IDKhuVuc = testAreaID
	residentRequest := validBatchRequest(RoleCodeResident, "resident@example.com")

	created, err := service.CreateBatch(
		context.Background(),
		testActorID,
		RoleCodeOwner,
		testBuildingID,
		[]BatchUserRequest{residentRequest, managerRequest},
	)
	if err != nil {
		t.Fatalf("CreateBatch() error = %v", err)
	}
	if repository.transaction != 1 || repository.committed != 1 ||
		repository.rolledBack != 0 || repository.findOrCreate != 2 || len(created) != 2 {
		t.Fatalf("transaction/repository calls = %+v, created=%d", repository, len(created))
	}
	if buildingOwner.calls != 1 || buildingOwner.building != testBuildingID ||
		buildingOwner.owner != testActorID {
		t.Fatalf("owner validation calls = %+v, want one check for requested building and actor", buildingOwner)
	}
	if len(repository.created) != 2 ||
		repository.created[0].MatKhauHash != "hashed:password-123" {
		t.Fatalf("prepared accounts = %+v", repository.created)
	}

	if memberships.calls != 1 || len(memberships.inputs) != 1 {
		t.Fatalf("membership AddBatch calls = %d, inputs=%+v", memberships.calls, memberships.inputs)
	}
	memberBatch := memberships.inputs[0]
	if memberBatch.BuildingID != testBuildingID ||
		len(memberBatch.Members) != 1 ||
		memberBatch.Members[0].AreaID != testAreaID ||
		memberBatch.Members[0].UserID != created[0].IDNguoiDung {
		t.Fatalf("membership batch = %+v", memberBatch)
	}

	if assignments.calls != 1 || len(assignments.inputs) != 1 {
		t.Fatalf("assignment AddBatch calls = %d, inputs=%+v", assignments.calls, assignments.inputs)
	}
	assignmentBatch := assignments.inputs[0]
	if assignmentBatch.BuildingID != testBuildingID ||
		len(assignmentBatch.Assignments) != 1 ||
		assignmentBatch.Assignments[0].FloorID != testFloorID ||
		assignmentBatch.Assignments[0].AreaID != testAreaID {
		t.Fatalf("assignment batch = %+v", assignmentBatch)
	}
}

func TestServiceAllowsAdminWithoutOwnerCheckAndRequiresBuildingID(t *testing.T) {
	repository := &fakeBatchUserRepository{
		roles: map[string]VaiTro{
			"code:CU_DAN": {
				IDVaiTro:     "550e8400-e29b-41d4-a716-446655440001",
				KyHieuVaiTro: RoleCodeResident,
			},
		},
	}
	memberships := &fakeMembershipService{}
	buildingOwner := &fakeBuildingOwnerService{}
	service, err := NewService(
		repository,
		memberships,
		&fakeAssignmentService{},
		buildingOwner,
		func(password string) (string, error) { return "hashed:" + password, nil },
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	_, err = service.CreateBatch(
		context.Background(),
		testActorID,
		RoleCodeAdmin,
		testBuildingID,
		[]BatchUserRequest{validBatchRequest(RoleCodeResident, "resident@example.com")},
	)
	if err != nil {
		t.Fatalf("admin CreateBatch() error = %v", err)
	}
	if len(memberships.inputs) != 1 ||
		memberships.inputs[0].BuildingID != testBuildingID {
		t.Fatalf("admin membership batch = %+v", memberships.inputs)
	}
	if buildingOwner.calls != 0 {
		t.Fatalf("owner validation called for admin: %+v", buildingOwner)
	}

	_, err = service.CreateBatch(
		context.Background(),
		testActorID,
		RoleCodeAdmin,
		"",
		[]BatchUserRequest{validBatchRequest(RoleCodeResident, "other@example.com")},
	)
	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr.HTTPStatus != http.StatusBadRequest {
		t.Fatalf("missing building ID error = %v, want HTTP 400", err)
	}
}

func TestServiceRejectsOwnerBeforeCreatingBatchWhenNotBuildingOwner(t *testing.T) {
	repository := &fakeBatchUserRepository{
		roles: map[string]VaiTro{
			"code:CU_DAN": {
				IDVaiTro:     "550e8400-e29b-41d4-a716-446655440001",
				KyHieuVaiTro: RoleCodeResident,
			},
		},
	}
	memberships := &fakeMembershipService{}
	buildingOwner := &fakeBuildingOwnerService{
		err: apperror.New(
			http.StatusForbidden,
			apperror.AppErrorForbidden,
			"building is not owned by the authenticated owner",
			nil,
		),
	}
	service, err := NewService(
		repository,
		memberships,
		&fakeAssignmentService{},
		buildingOwner,
		func(password string) (string, error) { return "hashed:" + password, nil },
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, err = service.CreateBatch(
		context.Background(),
		testActorID,
		RoleCodeOwner,
		testBuildingID,
		[]BatchUserRequest{validBatchRequest(RoleCodeResident, "resident@example.com")},
	)
	if !apperror.IsCode(err, apperror.AppErrorForbidden) {
		t.Fatalf("CreateBatch() error = %v, want forbidden", err)
	}
	if buildingOwner.calls != 1 || repository.findOrCreate != 0 ||
		repository.rolledBack != 1 || repository.committed != 0 || memberships.calls != 0 {
		t.Fatalf(
			"unauthorized owner batch was not rejected before writes: owner=%+v repository=%+v memberships=%+v",
			buildingOwner,
			repository,
			memberships,
		)
	}
}

func TestServiceRequiresBatchBuildingID(t *testing.T) {
	repository := &fakeBatchUserRepository{
		roles: map[string]VaiTro{
			"code:QUAN_LY": {
				IDVaiTro:     "550e8400-e29b-41d4-a716-446655440004",
				KyHieuVaiTro: RoleCodeManager,
			},
		},
	}
	service := newTestService(t, repository, &fakeMembershipService{}, &fakeAssignmentService{})
	request := validBatchRequest(RoleCodeManager, "manager@example.com")
	_, err := service.CreateBatch(
		context.Background(),
		testActorID,
		RoleCodeOwner,
		"",
		[]BatchUserRequest{request},
	)
	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr.HTTPStatus != http.StatusBadRequest {
		t.Fatalf("missing batch building ID error = %v, want HTTP 400", err)
	}
	if repository.transaction != 0 {
		t.Fatal("transaction started without a batch building ID")
	}
}

func TestServiceDoesNotMutateExistingAccountAndUsesItsIdentity(t *testing.T) {
	email := "user@example.com"
	repository := &fakeBatchUserRepository{
		roles: map[string]VaiTro{
			"code:CU_DAN": {
				IDVaiTro:     "550e8400-e29b-41d4-a716-446655440001",
				KyHieuVaiTro: RoleCodeResident,
			},
		},
		existing: map[string]CreatedUser{
			email: {
				NguoiDung: NguoiDung{
					IDNguoiDung:  "550e8400-e29b-41d4-a716-446655440011",
					IDVaiTro:     "550e8400-e29b-41d4-a716-446655440001",
					TenNguoiDung: "Existing name",
					Email:        email,
					TrangThai:    HoatDong,
				},
				KyHieuVaiTro: RoleCodeResident,
			},
		},
	}
	memberships := &fakeMembershipService{}
	service := newTestService(t, repository, memberships, &fakeAssignmentService{})
	result, err := service.CreateBatch(
		context.Background(),
		testActorID,
		RoleCodeOwner,
		testBuildingID,
		[]BatchUserRequest{validBatchRequest(RoleCodeResident, email)},
	)
	if err != nil {
		t.Fatalf("CreateBatch() error = %v", err)
	}
	if len(repository.created) != 0 {
		t.Fatalf("existing account was recreated or mutated: %+v", repository.created)
	}
	if len(result) != 1 ||
		result[0].IDNguoiDung != "550e8400-e29b-41d4-a716-446655440011" ||
		result[0].TenNguoiDung != "Existing name" {
		t.Fatalf("result did not preserve existing account: %+v", result)
	}
	if len(memberships.inputs) != 1 ||
		memberships.inputs[0].Members[0].UserID != "550e8400-e29b-41d4-a716-446655440011" {
		t.Fatalf("membership was not linked to existing user: %+v", memberships.inputs)
	}
}

func TestServiceRejectsExistingUserWithDifferentGlobalRole(t *testing.T) {
	email := "manager@example.com"
	repository := &fakeBatchUserRepository{
		roles: map[string]VaiTro{
			"code:QUAN_LY": {
				IDVaiTro:     "550e8400-e29b-41d4-a716-446655440004",
				KyHieuVaiTro: RoleCodeManager,
			},
		},
		existing: map[string]CreatedUser{
			email: {
				NguoiDung:    NguoiDung{IDNguoiDung: testActorID, Email: email},
				KyHieuVaiTro: RoleCodeResident,
			},
		},
	}
	service := newTestService(t, repository, &fakeMembershipService{}, &fakeAssignmentService{})
	_, err := service.CreateBatch(
		context.Background(),
		testActorID,
		RoleCodeOwner,
		testBuildingID,
		[]BatchUserRequest{validBatchRequest(RoleCodeManager, email)},
	)
	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr.HTTPStatus != http.StatusConflict {
		t.Fatalf("CreateBatch() error = %v, want HTTP 409", err)
	}
	if repository.rolledBack != 1 || repository.committed != 0 {
		t.Fatalf("role mismatch was not rolled back: %+v", repository)
	}
}

func TestServiceRollsBackWhenBatchRelationServiceFails(t *testing.T) {
	repository := &fakeBatchUserRepository{
		roles: map[string]VaiTro{
			"code:CU_DAN": {
				IDVaiTro:     "550e8400-e29b-41d4-a716-446655440001",
				KyHieuVaiTro: RoleCodeResident,
			},
		},
	}
	memberships := &fakeMembershipService{err: errors.New("membership failure")}
	service := newTestService(t, repository, memberships, &fakeAssignmentService{})
	_, err := service.CreateBatch(
		context.Background(),
		testActorID,
		RoleCodeOwner,
		testBuildingID,
		[]BatchUserRequest{validBatchRequest(RoleCodeResident, "resident@example.com")},
	)
	if err == nil {
		t.Fatal("CreateBatch() error = nil, want relation error")
	}
	if memberships.calls != 1 || repository.rolledBack != 1 || repository.committed != 0 {
		t.Fatalf("failed relation batch was not rolled back: %+v", repository)
	}
}

func TestServiceRejectsDuplicateEmailBeforeTransaction(t *testing.T) {
	repository := &fakeBatchUserRepository{
		roles: map[string]VaiTro{
			"code:CU_DAN": {
				IDVaiTro:     "550e8400-e29b-41d4-a716-446655440001",
				KyHieuVaiTro: RoleCodeResident,
			},
		},
	}
	service := newTestService(t, repository, &fakeMembershipService{}, &fakeAssignmentService{})
	_, err := service.CreateBatch(
		context.Background(),
		testActorID,
		RoleCodeOwner,
		testBuildingID,
		[]BatchUserRequest{
			validBatchRequest(RoleCodeResident, "user@example.com"),
			validBatchRequest(RoleCodeResident, " USER@example.com "),
		},
	)
	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr.HTTPStatus != http.StatusBadRequest {
		t.Fatalf("CreateBatch() error = %v, want HTTP 400", err)
	}
	if repository.transaction != 0 {
		t.Fatal("transaction started for duplicate batch emails")
	}
}

func TestServiceRequiresAuthorizedRoleAndValidBatchSize(t *testing.T) {
	tests := []struct {
		name     string
		actor    string
		requests []BatchUserRequest
		status   int
	}{
		{name: "unauthorized actor", actor: RoleCodeManager, requests: []BatchUserRequest{validBatchRequest(RoleCodeResident, "user@example.com")}, status: http.StatusForbidden},
		{name: "empty batch", actor: RoleCodeOwner, requests: nil, status: http.StatusBadRequest},
		{name: "oversized batch", actor: RoleCodeOwner, requests: make([]BatchUserRequest, MaxBatchUsers+1), status: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &fakeBatchUserRepository{}
			service := newTestService(t, repository, &fakeMembershipService{}, &fakeAssignmentService{})
			_, err := service.CreateBatch(
				context.Background(),
				testActorID,
				tt.actor,
				testBuildingID,
				tt.requests,
			)
			var appErr *apperror.Error
			if !errors.As(err, &appErr) || appErr.HTTPStatus != tt.status {
				t.Fatalf("CreateBatch() error = %v, want HTTP %d", err, tt.status)
			}
			if repository.transaction != 0 {
				t.Fatal("transaction started for rejected request")
			}
		})
	}
}
