package ctx_test

import (
	"context"
	"fmt"
	"github.com/TykTechnologies/tyk/ctx"
	"github.com/TykTechnologies/tyk/user"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"

	"github.com/TykTechnologies/tyk/apidef"
	"github.com/TykTechnologies/tyk/apidef/oas"

	"github.com/TykTechnologies/tyk/internal/uuid"
)

func TestGetSetNilRequestSession(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com", nil)

	var sessionData *user.SessionState = nil

	err := ctx.SetRequestSession(req, sessionData, true, false, false)
	if err == nil {
		t.Error("want error value 'setting a nil context SessionData', got nil error")
	} else if !strings.Contains(err.Error(), "setting a nil context SessionData") {
		t.Errorf("want error value 'setting a nil context SessionData', got '%v'", err)
	}

	var nilSession *user.SessionState
	nilSession, err = ctx.GetRequestSession(req)
	if err != nil && !strings.Contains(err.Error(), "session data does not yet exist for this request") {
		fmt.Printf("reached here")
		panic(err)
	}

	assert.Nil(t, nilSession)
}

func TestGetSetLegacyRequestSession(t *testing.T) {
	metadata := make(map[string]interface{})
	legacyMetadata := make(map[string]interface{})
	testId := uuid.New()
	metadata["TEST_ID"] = testId
	legacyMetadata["TEST_ID"] = testId

	v := struct {
		MetaData map[string]interface{} `json:"meta_data"`
	}{
		MetaData: legacyMetadata,
	}

	req := httptest.NewRequest("GET", "http://example.com", nil).WithContext(context.WithValue(context.Background(), ctx.SessionData, v))

	legacySession, err := ctx.GetRequestSession(req)
	if err != nil {
		panic(err)
	}

	assert.Equal(t, legacySession.MetaData, metadata)
}


// Test for GetDefinition
func TestGetDefinition(t *testing.T) {
	apiDef := &apidef.APIDefinition{
		APIID: uuid.New(),
	}

	req := httptest.NewRequest("GET", "http://example.com", nil)

	assert.Nil(t, ctx.GetDefinition(req))

	ctx.SetDefinition(req, apiDef)
	cloned := ctx.GetDefinition(req)

	assert.Equal(t, apiDef, cloned)
}

// Test for GetOASDefinition
func TestGetOASDefinition(t *testing.T) {
	oasDef := &oas.OAS{}
	oasDef.Info = &openapi3.Info{
		Title:   uuid.New(),
		Version: "1",
	}

	req := httptest.NewRequest("GET", "http://example.com", nil)

	assert.Nil(t, ctx.GetOASDefinition(req))

	ctx.SetOASDefinition(req, oasDef)
	cloned := ctx.GetOASDefinition(req)

	assert.Equal(t, oasDef, cloned)
}

// Benchmark for GetDefinition
func BenchmarkGetDefinition(b *testing.B) {
	apiDef := &apidef.APIDefinition{
		APIID: uuid.New(),
	}

	req := httptest.NewRequest("GET", "http://example.com", nil)

	ctx.SetDefinition(req, apiDef)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cloned := ctx.GetDefinition(req)
		assert.Equal(b, apiDef, cloned)
	}
}

// Benchmark for GetOASDefinition
func BenchmarkGetOASDefinition(b *testing.B) {
	oasDef := &oas.OAS{}
	oasDef.Info = &openapi3.Info{
		Title:   uuid.New(),
		Version: "1",
	}

	req := httptest.NewRequest("GET", "http://example.com", nil)

	ctx.SetOASDefinition(req, oasDef)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cloned := ctx.GetOASDefinition(req)
		assert.Equal(b, oasDef, cloned)
	}
}
