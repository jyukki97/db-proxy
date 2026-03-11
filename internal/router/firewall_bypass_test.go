package router

import "testing"

func TestFirewall_CTEBypass_Delete(t *testing.T) {
	query := `WITH cte AS (DELETE FROM users) SELECT 1;`
	cfg := FirewallConfig{
		Enabled:                 true,
		BlockDeleteWithoutWhere: true,
	}

	res := CheckFirewall(query, cfg)
	if !res.Blocked {
		t.Errorf("VULNERABLE: DELETE without WHERE hidden in CTE was allowed")
	}
}

func TestFirewall_CTEBypass_Update(t *testing.T) {
	query := `WITH cte AS (UPDATE users SET active = false) SELECT 1;`
	cfg := FirewallConfig{
		Enabled:                 true,
		BlockUpdateWithoutWhere: true,
	}

	res := CheckFirewall(query, cfg)
	if !res.Blocked {
		t.Errorf("VULNERABLE: UPDATE without WHERE hidden in CTE was allowed")
	}
}
