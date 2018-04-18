package graviteelb

import (
	"testing"

	"github.com/rancher/external-lb/model"
)

func TestGetConfigHash(t *testing.T) {
	config := model.LBConfig{
		LBEndpoint: "sadsdafdsaf",
		LBLabels: map[string]string{
			"kdjvtziugdbfn":  "ksljndbvfjhsdjhvbf",
			"8475vh687thvg":  "cb43t5634",
			"vpornihzube6g7": "73n4ct67b348v",
			"iu4bt367r5g34":  "783v46t48b34",
		},
		LBTargetPoolName: "sdjhfiu434",
		LBTargetPort:     "345",
		LBTargets: []model.LBTarget{
			model.LBTarget{HostIP: "234.234.223.4", Port: "7645"},
			model.LBTarget{HostIP: "234.9.223.4", Port: "44"},
		},
	}

	hash := getConfigHash(&config)

	t.Logf("Hash: %v", hash)

	for i := 0; i < 10; i++ {
		hash2 := getConfigHash(&config)

		if hash != hash2 {
			t.Fatal("Config hash is different")
		}
	}
}
