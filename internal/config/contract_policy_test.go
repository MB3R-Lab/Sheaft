package config

import (
	"strings"
	"testing"

	"github.com/MB3R-Lab/Sheaft/internal/modelcontract"
)

func TestContractPolicyEvaluate_AllowsSupportedContract(t *testing.T) {
	t.Parallel()

	policy := ContractPolicy{
		AllowedKinds: []string{"model", "snapshot"},
		AllowedContracts: []ContractPolicySelector{
			{
				Kind:     "snapshot",
				Name:     modelcontract.BeringSnapshotV100Name,
				Versions: []string{modelcontract.BeringSnapshotV100Version},
			},
		},
	}

	decision, err := policy.Evaluate(modelcontract.SupportedContract{
		Kind:    modelcontract.KindSnapshot,
		Name:    modelcontract.BeringSnapshotV100Name,
		Version: modelcontract.BeringSnapshotV100Version,
	})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if decision.Status != ContractPolicyStatusCurrent || decision.Action != string(ContractPolicyActionAllow) {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestContractPolicyEvaluate_WarnsOnDeprecatedContract(t *testing.T) {
	t.Parallel()

	policy := ContractPolicy{
		DeprecatedAction: ContractPolicyActionWarn,
		DeprecatedContracts: []DeprecatedContractSelector{
			{
				Kind:     "snapshot",
				Name:     modelcontract.BeringSnapshotV100Name,
				Versions: []string{modelcontract.BeringSnapshotV100Version},
			},
		},
	}

	decision, err := policy.Evaluate(modelcontract.SupportedContract{
		Kind:    modelcontract.KindSnapshot,
		Name:    modelcontract.BeringSnapshotV100Name,
		Version: modelcontract.BeringSnapshotV100Version,
	})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if decision.Status != ContractPolicyStatusDeprecated || decision.Action != string(ContractPolicyActionWarn) {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestContractPolicyEvaluate_FailsOnDisallowedKind(t *testing.T) {
	t.Parallel()

	policy := ContractPolicy{
		AllowedKinds: []string{"model"},
	}

	_, err := policy.Evaluate(modelcontract.SupportedContract{
		Kind:    modelcontract.KindSnapshot,
		Name:    modelcontract.BeringSnapshotV100Name,
		Version: modelcontract.BeringSnapshotV100Version,
	})
	if err == nil {
		t.Fatal("expected disallowed kind error")
	}
	if !strings.Contains(err.Error(), "allowed kinds") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContractPolicyValidate_RejectsUnknownSelector(t *testing.T) {
	t.Parallel()

	err := (ContractPolicy{
		AllowedContracts: []ContractPolicySelector{
			{
				Kind:     "model",
				Name:     modelcontract.BeringModelV100Name,
				Versions: []string{"9.9.9"},
			},
		},
	}).Validate()
	if err == nil {
		t.Fatal("expected validation error for unknown supported version")
	}
}
