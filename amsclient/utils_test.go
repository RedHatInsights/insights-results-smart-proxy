// Copyright 2021, 2022 Red Hat, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package amsclient_test

import (
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	ocmtesting "github.com/openshift-online/ocm-sdk-go/testing"
)

func MakeTokenObject(claims jwt.MapClaims) *jwt.Token {
	merged := jwt.MapClaims{}
	for name, value := range ocmtesting.MakeClaims() {
		merged[name] = value
	}
	for name, value := range claims {
		if value == nil {
			delete(merged, name)
		} else {
			merged[name] = value
		}
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, merged)
	token.Header["kid"] = "123"

	var err error
	token.Raw, err = token.SignedString(jwtPrivateKey)
	if err != nil {
		panic(err)
	}
	return token
}

func MakeTokenString(typ string, life time.Duration) string {
	token := MakeTokenObject(jwt.MapClaims{
		"typ": typ,
		"exp": time.Now().Add(life).Unix(),
	})

	return token.Raw
}

// Public key in PEM format:
const jwtPublicKeyPEM = `
-----BEGIN PUBLIC KEY-----
MIIBojANBgkqhkiG9w0BAQEFAAOCAY8AMIIBigKCAYEAny/9eLVyvEdOkY5ciQGr
2CAnukSK1MIwNd5re4Im8CfJzqqRREE6cXcbwj0ktcs/h+ahZdx0VbRQVPIs0oPI
zEYKlhtufxdeq+V2HL+2zILkesY7IL9N0lE4zhj7AnJl3MNu1Ur25StTmqK8b89L
lINH1DcCARTLhgKYGAUuWsxGYLRZGDfV2bjeAD7ZK8Zk9mPzr9a+SRTsj5XlcGqt
R6Vh52KFwW1eW13pXaKjOECN15k6AbYVPvRgwxSiK8PzxyFhw+OgtPT8FmjY3I42
uA47zsD0FbrT2RQ2nNAjsaJQtE61oSfuDCSiJMxytt/kwQd5xzb0XVR5rnJP2X4H
GSCre9tEN4IB/p/35EcVnEEJ9SaRUxMO7l7/Wv8Ia255xiqLuvK5T14ucMrjoMxG
kKDhN4NDutQYIBO8a0EPtJeC3/6pgXJadgIbI/5HBTay0b0b6Bco9gtKZN1crpWW
LABinXKMFZUVmfUmCKruUyqkTmuCXgpFWMjVpF+/o5c5AgMBAAE=
-----END PUBLIC KEY-----
`

// Private key in PEM format:
const jwtPrivateKeyPEM = `
-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----
`
