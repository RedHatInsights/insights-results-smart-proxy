/*
Copyright © 2024 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package auth_test

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RedHatInsights/insights-results-smart-proxy/auth"
	types "github.com/RedHatInsights/insights-results-types"
	"github.com/stretchr/testify/assert"
)

func TestGetAuthTokenHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(auth.XRHAuthTokenHeader, "test-token")

	token := auth.GetAuthTokenHeader(req)
	assert.Equal(t, "test-token", token)

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	token = auth.GetAuthTokenHeader(req)
	assert.Empty(t, token)
}

func TestDecodeTokenFromHeader_ValidToken(t *testing.T) {
	validToken := base64.StdEncoding.EncodeToString([]byte(`{
		"identity": {
			"account_number": "13043",
			"org_id": "1",
			"type": "User",
			"auth_type": "jwt-auth",
			"user": {
				"user_id": "1"
			}
		},
		"entitlements": {}
	}`))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(auth.XRHAuthTokenHeader, validToken)

	token, err := auth.DecodeTokenFromHeader(httptest.NewRecorder(), req, "xrh")
	assert.NoError(t, err)
	assert.NotNil(t, token)
	assert.Equal(t, types.UserID("13043"), token.Identity.AccountNumber)
	assert.Equal(t, types.OrgID(1), token.Identity.OrgID)
	assert.Equal(t, types.UserID("1"), token.Identity.User.UserID)
}

func TestDecodeTokenFromHeader_MissingToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	token, err := auth.DecodeTokenFromHeader(httptest.NewRecorder(), req, "xrh")
	assert.Error(t, err)
	assert.Nil(t, token)
	assert.Equal(t, auth.MissingTokenMessage, err.(*auth.AuthenticationError).ErrString)
}

func TestDecodeTokenFromHeader_MalformedToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(auth.XRHAuthTokenHeader, "invalid-base64")

	token, err := auth.DecodeTokenFromHeader(httptest.NewRecorder(), req, "xrh")
	assert.Error(t, err)
	assert.Nil(t, token)
	assert.Equal(t, auth.InvalidTokenMessage, err.(*auth.AuthenticationError).ErrString)
}

func TestGetAuthToken_ValidContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	identity := types.Identity{AccountNumber: "1", OrgID: 1, User: types.User{UserID: "1"}}
	ctx := context.WithValue(req.Context(), types.ContextKeyUser, identity)
	req = req.WithContext(ctx)

	result, err := auth.GetAuthToken(req)
	assert.NoError(t, err)
	assert.Equal(t, types.UserID("1"), result.AccountNumber)
	assert.Equal(t, types.UserID("1"), result.User.UserID) // Compare directly with types.UserID
}

func TestGetAuthToken_MissingContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	result, err := auth.GetAuthToken(req)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "token is not provided", err.(*auth.AuthenticationError).ErrString)
}

func TestGetAuthToken_InvalidContextType(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), types.ContextKeyUser, "invalid-type")
	req = req.WithContext(ctx)

	result, err := auth.GetAuthToken(req)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "contextKeyUser has wrong type", err.(*auth.AuthenticationError).ErrString)
}

func TestGetAuthTokenXRHHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth.GetAuthTokenHeader(r)
	})

	request, err := http.NewRequest(http.MethodGet, "an url", http.NoBody)
	assert.NoError(t, err)
	request.Header.Set(auth.XRHAuthTokenHeader, "token")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	// if xrh auth type is used, contents of header are checked in different function, thus only calling GetAuthTokenHeader is successful
	assert.Equal(t, http.StatusOK, recorder.Code)
}
