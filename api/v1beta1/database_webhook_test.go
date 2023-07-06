package v1beta1_test

import (
	"fmt"
	"testing"

	"github.com/db-operator/db-operator/api/v1beta1"
	"github.com/stretchr/testify/assert"
)

func TestSecretTemplatesValidator(t *testing.T) {
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
