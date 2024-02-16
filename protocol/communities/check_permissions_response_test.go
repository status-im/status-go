package communities

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/status-im/status-go/protocol/protobuf"
)

func TestCalculateRolesAndHighestRole(t *testing.T) {
	testCases := []struct {
		name                string
		permissions         map[string]*PermissionTokenCriteriaResult
		expectedRolesOrder  []protobuf.CommunityTokenPermission_Type
		expectedHighestRole protobuf.CommunityTokenPermission_Type
	}{
		{
			name: "Basic scenario with multiple permissions",
			permissions: map[string]*PermissionTokenCriteriaResult{
				"1": {
					Role: protobuf.CommunityTokenPermission_BECOME_MEMBER,
					TokenRequirements: []TokenRequirementResponse{
						{Satisfied: true},
					},
				},
				"2": {
					Role: protobuf.CommunityTokenPermission_BECOME_ADMIN,
					TokenRequirements: []TokenRequirementResponse{
						{Satisfied: true},
					},
				},
				"3": {
					Role: protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER,
					TokenRequirements: []TokenRequirementResponse{
						{Satisfied: false},
					},
				},
				"4": {
					Role: protobuf.CommunityTokenPermission_BECOME_TOKEN_OWNER,
					TokenRequirements: []TokenRequirementResponse{
						{Satisfied: false},
					},
				},
			},
			expectedRolesOrder:  []protobuf.CommunityTokenPermission_Type{protobuf.CommunityTokenPermission_BECOME_TOKEN_OWNER, protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER, protobuf.CommunityTokenPermission_BECOME_ADMIN, protobuf.CommunityTokenPermission_BECOME_MEMBER},
			expectedHighestRole: protobuf.CommunityTokenPermission_BECOME_ADMIN,
		},
		{
			name: "No member permission created",
			permissions: map[string]*PermissionTokenCriteriaResult{
				"2": {
					Role: protobuf.CommunityTokenPermission_BECOME_ADMIN,
					TokenRequirements: []TokenRequirementResponse{
						{Satisfied: false},
					},
				},
				"3": {
					Role: protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER,
					TokenRequirements: []TokenRequirementResponse{
						{Satisfied: false},
					},
				},
				"4": {
					Role: protobuf.CommunityTokenPermission_BECOME_TOKEN_OWNER,
					TokenRequirements: []TokenRequirementResponse{
						{Satisfied: false},
					},
				},
			},
			expectedRolesOrder:  []protobuf.CommunityTokenPermission_Type{protobuf.CommunityTokenPermission_BECOME_TOKEN_OWNER, protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER, protobuf.CommunityTokenPermission_BECOME_ADMIN, protobuf.CommunityTokenPermission_BECOME_MEMBER},
			expectedHighestRole: protobuf.CommunityTokenPermission_BECOME_MEMBER,
		},
		{
			name: "no permission satisfied",
			permissions: map[string]*PermissionTokenCriteriaResult{
				"1": {
					Role: protobuf.CommunityTokenPermission_BECOME_MEMBER,
					TokenRequirements: []TokenRequirementResponse{
						{Satisfied: false},
					},
				},
				"2": {
					Role: protobuf.CommunityTokenPermission_BECOME_ADMIN,
					TokenRequirements: []TokenRequirementResponse{
						{Satisfied: false},
					},
				},
				"3": {
					Role: protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER,
					TokenRequirements: []TokenRequirementResponse{
						{Satisfied: false},
					},
				},
				"4": {
					Role: protobuf.CommunityTokenPermission_BECOME_TOKEN_OWNER,
					TokenRequirements: []TokenRequirementResponse{
						{Satisfied: false},
					},
				},
			},
			expectedRolesOrder: []protobuf.CommunityTokenPermission_Type{protobuf.CommunityTokenPermission_BECOME_TOKEN_OWNER, protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER, protobuf.CommunityTokenPermission_BECOME_ADMIN, protobuf.CommunityTokenPermission_BECOME_MEMBER},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := calculateRolesAndHighestRole(tc.permissions)
			var actualOrder []protobuf.CommunityTokenPermission_Type
			for _, r := range result.Roles {
				actualOrder = append(actualOrder, r.Role)
			}
			if tc.expectedHighestRole == 0 {
				assert.Nil(t, result.HighestRole)
			} else {
				assert.Equal(t, tc.expectedHighestRole, result.HighestRole.Role, "Highest role is not calculated as expected")
			}
			assert.Equal(t, tc.expectedRolesOrder, actualOrder, "Roles are not calculated as expected")
		})
	}
}
