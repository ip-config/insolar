/*
 * The Clear BSD License
 *
 * Copyright (c) 2019 Insolar Technologies
 *
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without modification, are permitted (subject to the limitations in the disclaimer below) provided that the following conditions are met:
 *
 *  Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.
 *  Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.
 *  Neither the name of Insolar Technologies nor the names of its contributors may be used to endorse or promote products derived from this software without specific prior written permission.
 *
 * NO EXPRESS OR IMPLIED LICENSES TO ANY PARTY'S PATENT RIGHTS ARE GRANTED BY THIS LICENSE. THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 *
 */

package merkle

import (
	"context"
	"crypto"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/insolar/insolar/component"
	"github.com/insolar/insolar/core"
	"github.com/insolar/insolar/platformpolicy"
	"github.com/insolar/insolar/pulsar/pulsartestutils"
	"github.com/insolar/insolar/testutils"
	"github.com/insolar/insolar/testutils/nodekeeper"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func (t *calculatorHashesSuite) TestGetPulseHash() {
	pulseEntry := &PulseEntry{Pulse: t.pulse}
	ph, _, err := t.calculator.GetPulseProof(pulseEntry)
	t.Assert().NoError(err)

	expectedHash, _ := hex.DecodeString(
		"bd18c009950389026c5c6f85c838b899d188ec0d667f77948aa72a49747c3ed31835b1bdbb8bd1d1de62846b5f308ae3eac5127c7d36d7d5464985004122cc90",
	)

	t.Assert().Equal(OriginHash(expectedHash), ph)
}

func (t *calculatorHashesSuite) TestGetGlobuleHash() {
	pulseEntry := &PulseEntry{Pulse: t.pulse}
	ph, pp, err := t.calculator.GetPulseProof(pulseEntry)
	t.Assert().NoError(err)

	prevCloudHash, _ := hex.DecodeString(
		"00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
	)

	globuleEntry := &GlobuleEntry{
		PulseEntry: pulseEntry,
		PulseHash:  ph,
		ProofSet: map[core.Node]*PulseProof{
			t.nodeNetwork.GetOrigin(): pp,
		},
		PrevCloudHash: prevCloudHash,
		GlobuleID:     0,
	}
	gh, _, err := t.calculator.GetGlobuleProof(globuleEntry)
	t.Assert().NoError(err)

	expectedHash, _ := hex.DecodeString(
		"68cd36762548acd48795678c2e308978edd1ff74de2f5daf0511c1b52cf7a7bef44e09d5dd5806e99aa4ed4253aca88390e6b376e0c5f5a49ff48a8f9547e5c5",
	)

	t.Assert().Equal(OriginHash(expectedHash), gh)
}

func (t *calculatorHashesSuite) TestGetCloudHash() {
	pulseEntry := &PulseEntry{Pulse: t.pulse}
	ph, pp, err := t.calculator.GetPulseProof(pulseEntry)
	t.Assert().NoError(err)

	prevCloudHash, _ := hex.DecodeString(
		"00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
	)

	globuleEntry := &GlobuleEntry{
		PulseEntry: pulseEntry,
		PulseHash:  ph,
		ProofSet: map[core.Node]*PulseProof{
			t.nodeNetwork.GetOrigin(): pp,
		},
		PrevCloudHash: prevCloudHash,
		GlobuleID:     0,
	}
	_, gp, err := t.calculator.GetGlobuleProof(globuleEntry)

	ch, _, err := t.calculator.GetCloudProof(&CloudEntry{
		ProofSet:      []*GlobuleProof{gp},
		PrevCloudHash: prevCloudHash,
	})

	t.Assert().NoError(err)

	expectedHash, _ := hex.DecodeString(
		"68cd36762548acd48795678c2e308978edd1ff74de2f5daf0511c1b52cf7a7bef44e09d5dd5806e99aa4ed4253aca88390e6b376e0c5f5a49ff48a8f9547e5c5",
	)

	fmt.Println(hex.EncodeToString(ch))

	t.Assert().Equal(OriginHash(expectedHash), ch)
}

type calculatorHashesSuite struct {
	suite.Suite

	pulse       *core.Pulse
	nodeNetwork core.NodeNetwork
	service     core.CryptographyService

	calculator Calculator
}

func TestCalculatorHashes(t *testing.T) {
	calculator := &calculator{}

	key, _ := platformpolicy.NewKeyProcessor().GeneratePrivateKey()
	require.NotNil(t, key)

	service := testutils.NewCryptographyServiceMock(t)
	service.SignFunc = func(p []byte) (r *core.Signature, r1 error) {
		signature := core.SignatureFromBytes([]byte("signature"))
		return &signature, nil
	}
	service.GetPublicKeyFunc = func() (r crypto.PublicKey, r1 error) {
		return "key", nil
	}

	am := testutils.NewArtifactManagerMock(t)
	am.StateFunc = func() (r []byte, r1 error) {
		return []byte("state"), nil
	}

	scheme := platformpolicy.NewPlatformCryptographyScheme()
	nk := nodekeeper.GetTestNodekeeper(service)

	cm := component.Manager{}
	cm.Inject(nk, am, calculator, service, scheme)

	require.NotNil(t, calculator.ArtifactManager)
	require.NotNil(t, calculator.NodeNetwork)
	require.NotNil(t, calculator.CryptographyService)
	require.NotNil(t, calculator.PlatformCryptographyScheme)

	err := cm.Init(context.Background())
	require.NoError(t, err)

	pulse := &core.Pulse{
		PulseNumber:     core.PulseNumber(1337),
		NextPulseNumber: core.PulseNumber(1347),
		Entropy:         pulsartestutils.MockEntropyGenerator{}.GenerateEntropy(),
	}

	s := &calculatorHashesSuite{
		Suite:       suite.Suite{},
		calculator:  calculator,
		pulse:       pulse,
		nodeNetwork: nk,
		service:     service,
	}
	suite.Run(t, s)
}
