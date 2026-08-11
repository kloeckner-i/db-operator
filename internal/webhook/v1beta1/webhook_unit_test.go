package v1beta1_test

import (
	"fmt"
	"testing"

	"github.com/db-operator/db-operator/v2/api/v1beta1"
	webhook "github.com/db-operator/db-operator/v2/internal/webhook/v1beta1"
	"github.com/db-operator/db-operator/v2/pkg/consts"
	"github.com/stretchr/testify/assert"
)

/*
	TODO: I've moved all the tests to one file while fixing errors
	      produced by breaking changes in the controller runtime
				Once we have a new API version, the tests will be moved
				there with a proper format
*/
// Common tests
func TestUnitTemplatesValidator(t *testing.T) {
	validTemplates := v1beta1.Templates{
		{Name: "TEMPLATE_1", Template: "{{ .Protocol }} {{ .Hostname }} {{ .Port }} {{ .Username }} {{ .Password }} {{ .Database }}"},
		{Name: "TEMPLATE_2", Template: "{{.Protocol }}"},
		{Name: "TEMPLATE_3", Template: "{{.Protocol }}"},
		{Name: "TEMPLATE_4", Template: "{{.Protocol}}"},
		{Name: "TEMPLATE_5", Template: "jdbc:{{ .Protocol }}://{{ .Username }}:{{ .Password }}@{{ .Hostname }}:{{ .Port }}/{{ .Database }}"},
		{Name: "TEMPLATE_6", Template: "{{ .Secret \"CHECK\" }}"},
		{Name: "TEMPLATE_7", Template: "{{ .ConfigMap \"CHECK\" }}"},
		{Name: "TEMPLATE_8", Template: "{{ .Query \"CHECK\" }}"},
		{Name: "TEMPLATE_9", Template: "{{ if eq 1 1 }} It's true {{ else }} It's false {{ end }}"},
		{Name: "TEMPLATE_10", Template: "{{ .InstanceVar \"TEST\" }}"},
	}

	err := webhook.ValidateTemplates(validTemplates, true)
	assert.NoErrorf(t, err, "expected no error %v", err)

	invalidTemplates := v1beta1.Templates{
		{Name: "TEMPLATE_1", Template: "{{ .InvalidField }}"},
		{Name: "TEMPLATE_2", Template: "{{ .Secret invalid }}"},
		{Name: "TEMPLATE_3", Template: "{{ .Secret }}"},
	}

	err = webhook.ValidateTemplates(invalidTemplates, true)
	assert.Errorf(t, err, "should get error %v", err)

	cmTemplates := v1beta1.Templates{
		{Name: "TEMPLATE_1", Template: "configmap template", Secret: false},
	}

	err = webhook.ValidateTemplates(cmTemplates, true)
	assert.NoErrorf(t, err, "expected no error: %v", err)
	err = webhook.ValidateTemplates(cmTemplates, false)
	assert.ErrorContains(t, err, "ConfigMap templating is not allowed for that kind. Please set .secret to true")
}

// Database tests
func TestUnitSecretTemplatesValidator(t *testing.T) {
	validTemplates := map[string]string{
		"TEMPLATE_1": "{{ .Protocol }} {{ .DatabaseHost }} {{ .DatabasePort }} {{ .UserName }} {{ .Password }} {{ .DatabaseName}}",
		"TEMPLATE_2": "{{.Protocol }}",
		"TEMPLATE_3": "{{.Protocol }}",
		"TEMPLATE_4": "{{.Protocol}}",
		"TEMPLATE_5": "jdbc:{{ .Protocol }}://{{ .UserName }}:{{ .Password }}@{{ .DatabaseHost }}:{{ .DatabasePort }}/{{ .DatabaseName }}",
	}

	err := webhook.ValidateSecretTemplates(validTemplates)
	assert.NoErrorf(t, err, "expected no error %v", err)

	invalidField := ".InvalidField"
	invalidTemplates := map[string]string{
		"TEMPLATE_1": fmt.Sprintf("{{ %s }}", invalidField),
	}

	err = webhook.ValidateSecretTemplates(invalidTemplates)
	assert.Errorf(t, err, "should get error %v", err)
	assert.Contains(t, err.Error(), invalidField, "the error doesn't contain expected substring")
	assert.Contains(t, err.Error(),
		"[.Protocol .DatabaseHost .DatabasePort .UserName .Password .DatabaseName]",
		"the error doesn't contain expected substring",
	)
}

// DbUser tests
func TestExtraPrivilegesFail1(t *testing.T) {
	privileges := []string{consts.ALL_PRIVILEGES}
	err := webhook.TestExtraPrivileges(privileges)
	assert.ErrorContains(t, err, "it's not allowed to grant ALL PRIVILEGES")
}

func TestExtraPrivilegesFail2(t *testing.T) {
	privileges := []string{"all privileges"}
	err := webhook.TestExtraPrivileges(privileges)
	assert.ErrorContains(t, err, "it's not allowed to grant ALL PRIVILEGES")
}

func TestExtraPrivilegesFail3(t *testing.T) {
	privileges := []string{"aLL PriVileges"}
	err := webhook.TestExtraPrivileges(privileges)
	assert.ErrorContains(t, err, "it's not allowed to grant ALL PRIVILEGES")
}

func TestExtraPrivileges(t *testing.T) {
	privileges := []string{"rds_admin"}
	err := webhook.TestExtraPrivileges(privileges)
	assert.NoError(t, err)
}

// DbInstance Tests

func TestUnitEngineValid(t *testing.T) {
	err := webhook.ValidateEngine("postgres")
	assert.NoError(t, err)

	err = webhook.ValidateEngine("mysql")
	assert.NoError(t, err)
}

func TestUnitEngineInvalid(t *testing.T) {
	err := webhook.ValidateEngine("dummy")
	assert.Error(t, err)
}

func TestValidateFromSecret(t *testing.T) {
	from := &v1beta1.FromRef{
		Kind: "Secret",
		Name: "name",
		Key:  "key",
	}
	dbin := &v1beta1.GenericInstance{
		HostFrom:     from,
		PortFrom:     from,
		PublicIPFrom: from,
	}
	assert.NoError(t, webhook.ValidateConfigFrom(dbin))
}

func TestValidateFromCM(t *testing.T) {
	from := &v1beta1.FromRef{
		Kind: "ConfigMap",
		Name: "name",
		Key:  "key",
	}
	dbin := &v1beta1.GenericInstance{
		HostFrom:     from,
		PortFrom:     from,
		PublicIPFrom: from,
	}
	assert.NoError(t, webhook.ValidateConfigFrom(dbin))
}

func TestValidateFromUnknown(t *testing.T) {
	from := &v1beta1.FromRef{
		Kind: "dummy",
		Name: "name",
		Key:  "key",
	}
	dbin := &v1beta1.GenericInstance{
		HostFrom:     from,
		PortFrom:     from,
		PublicIPFrom: from,
	}
	assert.Error(t, webhook.ValidateConfigFrom(dbin))
}

func TestUnitConfigHostErr(t *testing.T) {
	spec := &v1beta1.GenericInstance{
		Host: "host",
		HostFrom: &v1beta1.FromRef{
			Kind: "ConfigMap",
			Name: "name",
			Key:  "key",
		},
	}
	assert.Error(t, webhook.ValidateConfigVsConfigFrom(spec))
}

func TestUnitConfigHostVal(t *testing.T) {
	spec := &v1beta1.GenericInstance{
		Host: "host",
	}
	assert.NoError(t, webhook.ValidateConfigVsConfigFrom(spec))
}

func TestUnitConfigHostFrom(t *testing.T) {
	spec := &v1beta1.GenericInstance{
		HostFrom: &v1beta1.FromRef{
			Kind: "ConfigMap",
			Name: "name",
			Key:  "key",
		},
	}
	assert.NoError(t, webhook.ValidateConfigVsConfigFrom(spec))
}

func TestUnitConfigPortErr(t *testing.T) {
	spec := &v1beta1.GenericInstance{
		Port: 5432,
		PortFrom: &v1beta1.FromRef{
			Kind: "ConfigMap",
			Name: "name",
			Key:  "key",
		},
	}
	assert.Error(t, webhook.ValidateConfigVsConfigFrom(spec))
}

func TestUnitConfigPortVal(t *testing.T) {
	spec := &v1beta1.GenericInstance{
		Port: 5432,
	}
	assert.NoError(t, webhook.ValidateConfigVsConfigFrom(spec))
}

func TestUnitConfigPortFrom(t *testing.T) {
	spec := &v1beta1.GenericInstance{
		PortFrom: &v1beta1.FromRef{
			Kind: "ConfigMap",
			Name: "name",
			Key:  "key",
		},
	}
	assert.NoError(t, webhook.ValidateConfigVsConfigFrom(spec))
}

func TestUnitConfigPublicIPErr(t *testing.T) {
	spec := &v1beta1.GenericInstance{
		PublicIP: "123.123.123.123",
		PublicIPFrom: &v1beta1.FromRef{
			Kind: "ConfigMap",
			Name: "name",
			Key:  "key",
		},
	}
	assert.Error(t, webhook.ValidateConfigVsConfigFrom(spec))
}

func TestUnitConfigPublicIPVal(t *testing.T) {
	spec := &v1beta1.GenericInstance{
		PublicIP: "123.123.123.123",
	}
	assert.NoError(t, webhook.ValidateConfigVsConfigFrom(spec))
}

func TestUnitConfigPublicIPFrom(t *testing.T) {
	spec := &v1beta1.GenericInstance{
		PublicIPFrom: &v1beta1.FromRef{
			Kind: "ConfigMap",
			Name: "name",
			Key:  "key",
		},
	}
	assert.NoError(t, webhook.ValidateConfigVsConfigFrom(spec))
}

func TestAllowedPrivilegesFail1(t *testing.T) {
	privileges := []string{consts.ALL_PRIVILEGES}
	err := webhook.TestAllowedPrivileges(privileges)
	assert.ErrorContains(t, err, "it's not allowed to grant ALL PRIVILEGES")
}

func TestAllowedPrivilegesFail2(t *testing.T) {
	privileges := []string{"all privileges"}
	err := webhook.TestAllowedPrivileges(privileges)
	assert.ErrorContains(t, err, "it's not allowed to grant ALL PRIVILEGES")
}

func TestAllowedPrivilegesFail3(t *testing.T) {
	privileges := []string{"aLL PriVileges"}
	err := webhook.TestAllowedPrivileges(privileges)
	assert.ErrorContains(t, err, "it's not allowed to grant ALL PRIVILEGES")
}

func TestAllowedPrivileges(t *testing.T) {
	privileges := []string{"rds_admin"}
	err := webhook.TestAllowedPrivileges(privileges)
	assert.NoError(t, err)
}
