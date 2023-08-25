package v1beta1_test

import (
	"fmt"
	"testing"

	"github.com/db-operator/db-operator/api/v1beta1"
	"github.com/stretchr/testify/assert"
)

func TestUnitSecretTemplatesValidator(t *testing.T) {
	validTemplates := map[string]string{
		"TEMPLATE_1": "{{ .Protocol }} {{ .DatabaseHost }} {{ .DatabasePort }} {{ .UserName }} {{ .Password }} {{ .DatabaseName}}",
		"TEMPLATE_2": "{{.Protocol }}",
		"TEMPLATE_3": "{{.Protocol }}",
		"TEMPLATE_4": "{{.Protocol}}",
		"TEMPLATE_5": "jdbc:{{ .Protocol }}://{{ .UserName }}:{{ .Password }}@{{ .DatabaseHost }}:{{ .DatabasePort }}/{{ .DatabaseName }}",
	}

	err := v1beta1.ValidateSecretTemplates(validTemplates)
	assert.NoErrorf(t, err, "expected no error %v", err)

	invalidField := ".InvalidField"
	invalidTemplates := map[string]string{
		"TEMPLATE_1": fmt.Sprintf("{{ %s }}", invalidField),
	}

	err = v1beta1.ValidateSecretTemplates(invalidTemplates)
	assert.Errorf(t, err, "should get error %v", err)
	assert.Contains(t, err.Error(), invalidField, "the error doesn't contain expected substring")
	assert.Contains(t, err.Error(),
		"[.Protocol .DatabaseHost .DatabasePort .UserName .Password .DatabaseName]",
		"the error doesn't contain expected substring",
	)

}

func TestUnitTemplatesValidator(t *testing.T) {
	validTemplates := v1beta1.Templates{
		{ Name: "TEMPLATE_1", Template: "{{ .Protocol }} {{ .Host }} {{ .Port }} {{ .Username }} {{ .Password }} {{ .Database }}"},
		{ Name: "TEMPLATE_2", Template: "{{.Protocol }}"},
		{ Name: "TEMPLATE_3", Template: "{{.Protocol }}"},
		{ Name: "TEMPLATE_4", Template: "{{.Protocol}}"},
		{ Name: "TEMPLATE_5", Template: "jdbc:{{ .Protocol }}://{{ .Username }}:{{ .Password }}@{{ .Host }}:{{ .Port }}/{{ .Database }}"},
		{ Name: "TEMPLATE_6", Template: "{{ .Secret \"CHECK\" }}"},
		{ Name: "TEMPLATE_7", Template: "{{ .ConfigMap \"CHECK\" }}"},
		{ Name: "TEMPLATE_8", Template: "{{ .Query \"CHECK\" }}"},
		{ Name: "TEMPLATE_9", Template: "{{ if eq 1 1 }} It's true {{ else }} It's false {{ end }}"},
	}

	err := v1beta1.ValidateTemplates(validTemplates)
	assert.NoErrorf(t, err, "expected no error %v", err)

	invalidTemplates := v1beta1.Templates{
		{ Name: "TEMPLATE_1", Template: "{{ .InvalidField }}"},
		{ Name: "TEMPLATE_2", Template: "{{ .Secret invalid }}"},
	}

	err = v1beta1.ValidateTemplates(invalidTemplates)
	assert.Errorf(t, err, "should get error %v", err)
}
