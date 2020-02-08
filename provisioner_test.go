package main

import (
	"errors"
	"net/http"
	"testing"

	apiTester "github.com/finarfin/go-apiclient-tester/tester"
	"github.com/finarfin/go-apiclient-tester/postman"
	salt "github.com/finarfin/go-salt-netapi-client/cherrypy"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func testConfig(t *testing.T, c map[string]interface{}) *terraform.ResourceConfig {
	return terraform.NewResourceConfigRaw(c)
}

func setup(t *testing.T) *apiTester.Tester {
	tester, err := postman.NewTester("testdata/postman_collection.json")
	if err != nil {
		t.Fatal(err)
	}

	return tester
}

func TestResourceProvisioner_impl(t *testing.T) {
	var _ terraform.ResourceProvisioner = Provisioner()
}

func TestProvisioner(t *testing.T) {
	if err := Provisioner().(*schema.Provisioner).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestResourceProvider_Validate_good(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"address":   "http://example.com",
		"username":  "test_user",
		"password":  "test_pwd",
		"backend":   "pam",
		"minion_id": "sample_minion",
	})

	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) > 0 {
		t.Fatalf("Errors: %v", errs)
	}
}

// Happy path
func TestResourceProvider_GoodLogin_MinionOnline_StateSuccess(t *testing.T) {
	tester := setup(t)
	defer tester.Close()
	tester.Setup(t, "auth_login", "success")
	tester.Setup(t, "minions_get", "success")
	tester.Setup(t, "run", "online_success")

	c := testConfig(t, map[string]interface{}{
		"address":   tester.URL,
		"username":  "test_user",
		"password":  "test_pwd",
		"backend":   "pam",
		"minion_id": "minion1",
	})

	output := new(terraform.MockUIOutput)
	p := Provisioner()

	if err := p.Apply(output, nil, c); err != nil {
		t.Fatalf("Err: %s", err)
	}

	return
}

// Highstate failure
func TestResourceProvider_GoodLogin_MinionOnline_StateFail(t *testing.T) {
	tester := setup(t)
	defer tester.Close()
	tester.Setup(t, "auth_login", "success")
	tester.Setup(t, "minions_get", "success")
	tester.Setup(t, "run", "online_failed")

	c := testConfig(t, map[string]interface{}{
		"address":   tester.URL,
		"username":  "test_user",
		"password":  "test_pwd",
		"backend":   "pam",
		"minion_id": "minion1",
	})

	output := new(terraform.MockUIOutput)
	p := Provisioner()

	if err := p.Apply(output, nil, c); err != nil {
		if !errors.Is(err, ErrorHighstateFailed) {
			t.Fatalf("Unexpected error: %s", err)
		}

		return
	}

	t.Fatal("Expected state execution request to fail")
}

// Minion goes offline after wait before Highstate and goes online again
func TestResourceProvider_GoodLogin_MinionOnline_StateOfflineToOnline(t *testing.T) {
	tester := setup(t)
	defer tester.Close()
	tester.Setup(t, "auth_login", "success")
	tester.Setup(t, "minions_get", "success")

	offline, err := tester.Scenario("run", "offline")
	if err != nil {
		t.Fatal(err)
	}

	success, err := tester.Scenario("run", "online_success")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []*apiTester.TestScenario{offline, success}
	current := 0

	tester.Do(offline.Request.Path, func(w http.ResponseWriter, req *http.Request) {
		apiTester.WriteResponse(t, &scenarios[current].Response, w)
		current++
	})

	c := testConfig(t, map[string]interface{}{
		"address":   tester.URL,
		"username":  "test_user",
		"password":  "test_pwd",
		"backend":   "pam",
		"minion_id": "minion1",
	})

	output := new(terraform.MockUIOutput)
	p := Provisioner()

	if err := p.Apply(output, nil, c); err != nil {
		t.Fatalf("err: %s", err)
	}

	return
}

// Minion goes offline while state apply
func TestResourceProvider_GoodLogin_MinionOnline_StateOffline(t *testing.T) {
	tester := setup(t)
	defer tester.Close()
	tester.Setup(t, "auth_login", "success")
	tester.Setup(t, "minions_get", "success")
	tester.Setup(t, "run", "offline")

	c := testConfig(t, map[string]interface{}{
		"address":   tester.URL,
		"username":  "test_user",
		"password":  "test_pwd",
		"backend":   "pam",
		"minion_id": "minion1",
	})

	output := new(terraform.MockUIOutput)
	p := Provisioner()

	if err := p.Apply(output, nil, c); err != nil {
		if !errors.Is(err, ErrorMinionNotAvailable) {
			t.Fatalf("Unexpected error: %s", err)
		}

		return
	}

	t.Fatal("Expected state execution request to fail")
}

// Minion transitions from missing to offline to online
func TestResourceProvider_GoodLogin_MinionMissingOfflineSuccess_StateSuccess(t *testing.T) {
	tester := setup(t)
	defer tester.Close()
	tester.Setup(t, "auth_login", "success")
	tester.Setup(t, "run", "online_success")

	missing, err := tester.Scenario("minions_get", "missing")
	if err != nil {
		t.Fatal(err)
	}

	offline, err := tester.Scenario("minions_get", "offline")
	if err != nil {
		t.Fatal(err)
	}

	success, err := tester.Scenario("minions_get", "success")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []*apiTester.TestScenario{missing, offline, success}
	current := 0

	tester.Do("/minions/minion1", func(w http.ResponseWriter, req *http.Request) {
		apiTester.WriteResponse(t, &scenarios[current].Response, w)
		current++
	})

	c := testConfig(t, map[string]interface{}{
		"address":   tester.URL,
		"username":  "test_user",
		"password":  "test_pwd",
		"backend":   "pam",
		"minion_id": "minion1",
	})

	output := new(terraform.MockUIOutput)
	p := Provisioner()

	if err := p.Apply(output, nil, c); err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
}

// Invalid credentials
func TestResourceProvider_BadLogin(t *testing.T) {
	tester := setup(t)
	defer tester.Close()
	tester.Setup(t, "auth_login", "bad_user")

	c := testConfig(t, map[string]interface{}{
		"address":   tester.URL,
		"username":  "test_user",
		"password":  "test_pwd",
		"backend":   "pam",
		"minion_id": "sample_minion",
	})

	output := new(terraform.MockUIOutput)
	p := Provisioner()

	if err := p.Apply(output, nil, c); err != nil {
		if err != salt.ErrorInvalidCredentials {
			t.Fatalf("Unexpected error: %s", err)
		}

		return
	}

	t.Fatal("Expected login request to fail")
}
