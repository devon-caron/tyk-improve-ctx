package ctx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/TykTechnologies/tyk/apidef"
	"github.com/TykTechnologies/tyk/apidef/oas"
	"github.com/TykTechnologies/tyk/config"
	"github.com/TykTechnologies/tyk/internal/reflect"
	"github.com/TykTechnologies/tyk/internal/service/core"
	"github.com/TykTechnologies/tyk/storage"
	"github.com/TykTechnologies/tyk/user"

	logger "github.com/TykTechnologies/tyk/log"
)

type Key uint

const (
	SessionData Key = iota
	// Deprecated: UpdateSession was used to trigger a session update, use *SessionData.Touch instead.
	UpdateSession
	AuthToken
	HashedAuthToken
	VersionData
	VersionName
	VersionDefault
	OrgSessionContext
	ContextData
	RetainHost
	TrackThisEndpoint
	DoNotTrackThisEndpoint
	UrlRewritePath
	RequestMethod
	OrigRequestURL
	LoopLevel
	LoopLevelLimit
	ThrottleLevel
	ThrottleLevelLimit
	Trace
	CheckLoopLimits
	UrlRewriteTarget
	TransformedRequestMethod
	Definition
	RequestStatus
	GraphQLRequest
	GraphQLIsWebSocketUpgrade
	OASOperation

	// CacheOptions holds cache options required for cache writer middleware.
	CacheOptions
	OASDefinition
	SelfLooping
)

func ctxSetSession(r *http.Request, s *user.SessionState, scheduleUpdate bool, hashKey bool) {
	if s == nil {
		panic("setting a nil context SessionData")
	}

	if s.KeyID == "" {
		s.KeyID = GetAuthToken(r)
	}

	if s.KeyHashEmpty() {
		s.SetKeyHash(storage.HashKey(s.KeyID, hashKey))
	}

	ctx := r.Context()
	ctx = context.WithValue(ctx, SessionData, s)

	ctx = context.WithValue(ctx, AuthToken, s.KeyID)

	if scheduleUpdate {
		s.Touch()
	}

	core.SetContext(r, ctx)
}

func GetAuthToken(r *http.Request) string {
	if v := r.Context().Value(AuthToken); v != nil {
		value, ok := v.(string)
		if ok {
			return value
		}
	}
	return ""
}

// GetRequestSession will retrieve a reference to an existing request's session data.
// Returns an error if either session doesn't exist for the provided request, or if using a legacy
// SessionState type and marshalling errors occur.
func GetRequestSession(r *http.Request) (*user.SessionState, error) {
	v := r.Context().Value(SessionData)

	if v == nil {
		return nil, fmt.Errorf("session data does not yet exist for this request")
	}

	if val, ok := v.(*user.SessionState); ok {
		return val, nil
	} else {
		logger.Get().Warning("SessionState struct differ from the gateway version, trying to unmarshal.")

		sess := user.SessionState{}
		b, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("error marshalling legacy session data: %v", err)
		}

		err = json.Unmarshal(b, &sess)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling legacy session data: %v", err)
		}

		return &sess, nil
	}
}

// SetRequestSession sets s as the session data for a request. Signals the gateway to update
// the session data internally if scheduleUpdate is true. Optionally, specify whether to
// obfuscate/hash tokens in Redis. Returns an error if one is encountered while setting session state.
func SetRequestSession(r *http.Request, s *user.SessionState, scheduleUpdate bool, hashKey ...bool) error {
	if s == nil {
		return fmt.Errorf("error: attempted to set a nil context SessionData")
	}

	if len(hashKey) > 0 {
		ctxSetSession(r, s, scheduleUpdate, hashKey[0])
	} else if config.Global != nil {
		ctxSetSession(r, s, scheduleUpdate, config.Global().HashKeys)
	} else {
		logger.Get().Warnf("Global config not set, defaulting to no-hash")
		ctxSetSession(r, s, scheduleUpdate, false)
	}

	return nil
}

// GetSession will retrieve a reference to an existing request's session data, or nil if it does not exist.
func GetSession(r *http.Request) *user.SessionState {
	if v := r.Context().Value(SessionData); v != nil {
		if val, ok := v.(*user.SessionState); ok {
			return val
		} else {
			logger.Get().Warning("SessionState struct differ from the gateway version, trying to unmarshal.")
			sess := user.SessionState{}
			b, _ := json.Marshal(v)
			e := json.Unmarshal(b, &sess)
			if e == nil {
				return &sess
			}
		}
	}

	logger.Get().Warning("Empty session retrieved")
	return nil
}

// SetSession sets s as the session data for a request. Signals the gateway to update
// the session data internally if scheduleUpdate is true. Optionally, specify whether to
// obfuscate/hash tokens in Redis.
func SetSession(r *http.Request, s *user.SessionState, scheduleUpdate bool, hashKey ...bool) {
	if len(hashKey) > 1 {
		ctxSetSession(r, s, scheduleUpdate, hashKey[0])
	} else {
		ctxSetSession(r, s, scheduleUpdate, config.Global().HashKeys)
	}
}

// SetDefinition sets an API definition object to the request context.
func SetDefinition(r *http.Request, s *apidef.APIDefinition) {
	ctx := r.Context()
	ctx = context.WithValue(ctx, Definition, s)
	core.SetContext(r, ctx)
}

// GetDefinition will return a deep copy of the API definition valid for the request.
func GetDefinition(r *http.Request) *apidef.APIDefinition {
	if v := r.Context().Value(Definition); v != nil {
		if val, ok := v.(*apidef.APIDefinition); ok {
			return reflect.Clone(val)
		}
	}

	return nil
}

// SetOASDefinition sets an OAS API definition object to the request context.
func SetOASDefinition(r *http.Request, s *oas.OAS) {
	ctx := r.Context()
	ctx = context.WithValue(ctx, OASDefinition, s)
	core.SetContext(r, ctx)
}

// GetOASDefinition will return a deep copy of the OAS API definition valid for the request.
func GetOASDefinition(r *http.Request) *oas.OAS {
	if v := r.Context().Value(OASDefinition); v != nil {
		if val, ok := v.(*oas.OAS); ok {
			return reflect.Clone(val)
		}
	}

	return nil
}
