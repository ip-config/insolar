/*
 *    Copyright 2018 Insolar
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package servicenetwork

import (
	"bytes"
	"context"
	"crypto"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/insolar/insolar/certificate"
	"github.com/insolar/insolar/component"
	"github.com/insolar/insolar/configuration"
	"github.com/insolar/insolar/consensus/packets"
	"github.com/insolar/insolar/consensus/phases"
	"github.com/insolar/insolar/core"
	"github.com/insolar/insolar/cryptography"
	"github.com/insolar/insolar/log"
	"github.com/insolar/insolar/network"
	"github.com/insolar/insolar/network/merkle"
	"github.com/insolar/insolar/network/nodenetwork"
	"github.com/insolar/insolar/platformpolicy"
	"github.com/insolar/insolar/testutils"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type testSuite struct {
	suite.Suite
	ctx            context.Context
	bootstrapNodes []networkNode
	networkNodes   []networkNode
	testNode       networkNode
	networkPort    int
}

func NewTestSuite() *testSuite {
	return &testSuite{
		Suite:        suite.Suite{},
		ctx:          context.Background(),
		networkNodes: make([]networkNode, 0),
		networkPort:  10001,
	}
}

type PhaseTimeOut uint8

const (
	Disable = PhaseTimeOut(iota + 1)
	Partitial
	Full
)

func (s *testSuite) InitNodes() {
	for _, n := range s.bootstrapNodes {
		err := n.componentManager.Init(s.ctx)
		s.NoError(err)
	}
	log.Info("========== Bootstrap nodes inited")
	<-time.After(time.Second * 1)

	if s.testNode.componentManager != nil {
		err := s.testNode.componentManager.Init(s.ctx)
		s.NoError(err)
	}
}

func (s *testSuite) StartNodes() {
	for _, n := range s.bootstrapNodes {
		err := n.componentManager.Start(s.ctx)
		s.NoError(err)
	}
	log.Info("========== Bootstrap nodes started")
	<-time.After(time.Second * 1)

	if s.testNode.componentManager != nil {
		err := s.testNode.componentManager.Init(s.ctx)
		s.NoError(err)
		err = s.testNode.componentManager.Start(s.ctx)
		s.NoError(err)
	}

}

func (s *testSuite) StopNodes() {
	for _, n := range s.networkNodes {
		err := n.componentManager.Stop(s.ctx)
		s.NoError(err)
	}

	if s.testNode.componentManager != nil {
		err := s.testNode.componentManager.Stop(s.ctx)
		s.NoError(err)
	}
}

type networkNode struct {
	componentManager *component.Manager
	serviceNetwork   *ServiceNetwork
}

func initCertificate(t *testing.T, nodes []certificate.BootstrapNode, key crypto.PublicKey, ref core.RecordRef) *certificate.CertificateManager {
	proc := platformpolicy.NewKeyProcessor()
	publicKey, err := proc.ExportPublicKey(key)
	assert.NoError(t, err)
	bytes.NewReader(publicKey)

	type сertInfo map[string]interface{}
	j := сertInfo{
		"public_key": string(publicKey[:]),
	}

	data, err := json.Marshal(j)

	cert, err := certificate.ReadCertificateFromReader(key, proc, bytes.NewReader(data))
	cert.Reference = ref.String()
	assert.NoError(t, err)
	cert.BootstrapNodes = nodes
	mngr := certificate.NewCertificateManager(cert)
	return mngr
}

func initCrypto(t *testing.T, nodes []certificate.BootstrapNode, ref core.RecordRef) (*certificate.CertificateManager, core.CryptographyService) {
	key, err := platformpolicy.NewKeyProcessor().GeneratePrivateKey()
	assert.NoError(t, err)
	require.NotNil(t, key)
	cs := cryptography.NewKeyBoundCryptographyService(key)
	pubKey, err := cs.GetPublicKey()
	assert.NoError(t, err)
	mngr := initCertificate(t, nodes, pubKey, ref)

	return mngr, cs
}

func (s *testSuite) getBootstrapNodes(t *testing.T) []certificate.BootstrapNode {
	result := make([]certificate.BootstrapNode, 0)
	for _, b := range s.bootstrapNodes {
		node := certificate.NewBootstrapNode(
			b.serviceNetwork.CertificateManager.GetCertificate().GetPublicKey(),
			b.serviceNetwork.CertificateManager.GetCertificate().(*certificate.Certificate).PublicKey,
			b.serviceNetwork.cfg.Host.Transport.Address,
			b.serviceNetwork.NodeNetwork.GetOrigin().ID().String())
		result = append(result, *node)
	}
	return result
}

func (s *testSuite) createNetworkNode(t *testing.T, timeOut PhaseTimeOut) networkNode {
	address := "127.0.0.1:" + strconv.Itoa(s.networkPort)
	s.networkPort += 2 // coz consensus transport port+=1

	origin := nodenetwork.NewNode(testutils.RandomRef(),
		core.StaticRoleVirtual,
		nil,
		address,
		"",
	)

	cfg := configuration.NewConfiguration()
	cfg.Host.Transport.Address = address

	scheme := platformpolicy.NewPlatformCryptographyScheme()
	serviceNetwork, err := NewServiceNetwork(cfg, scheme)
	assert.NoError(t, err)

	pulseManagerMock := testutils.NewPulseManagerMock(t)
	netCoordinator := testutils.NewNetworkCoordinatorMock(t)
	netCoordinator.ValidateCertMock.Set(func(p context.Context, p1 core.AuthorizationCertificate) (bool, error) {
		return true, nil
	})

	amMock := testutils.NewArtifactManagerMock(t)

	certManager, cryptographyService := initCrypto(t, s.getBootstrapNodes(t), origin.ID())
	netSwitcher := testutils.NewNetworkSwitcherMock(t)

	var phaseManager phases.PhaseManager
	switch timeOut {
	case Disable:
		phaseManager = phases.NewPhaseManager()
	case Full:
		phaseManager = &FullTimeoutPhaseManager{}
	case Partitial:
		phaseManager = &PartitialTimeoutPhaseManager{}
	}

	realKeeper := nodenetwork.NewNodeKeeper(origin)
	keeper := &nodeKeeperWrapper{realKeeper}

	cm := &component.Manager{}
	cm.Register(keeper, pulseManagerMock, netCoordinator, amMock, realKeeper)
	cm.Register(certManager, cryptographyService, phaseManager)
	cm.Inject(serviceNetwork, netSwitcher)

	serviceNetwork.NodeKeeper = keeper

	return networkNode{cm, serviceNetwork}
}

func (s *testSuite) TestNodeConnect() {
	s.T().Skip("will be available after phase result fix !")
	phasesResult := make(chan error)
	bootstrapNode1 := s.createNetworkNode(s.T(), Disable)
	s.bootstrapNodes = append(s.bootstrapNodes, bootstrapNode1)

	s.testNode = s.createNetworkNode(s.T(), Disable)

	s.InitNodes()
	s.StartNodes()
	res := <-phasesResult
	s.NoError(res)
	activeNodes := s.testNode.serviceNetwork.NodeKeeper.GetActiveNodes()
	s.Equal(2, len(activeNodes))
	// teardown
	<-time.After(time.Second * 5)
	s.StopNodes()
}

func TestServiceNetworkIntegration(t *testing.T) {
	s := NewTestSuite()
	suite.Run(t, s)
}

// Full timeout test

type FullTimeoutPhaseManager struct {
}

func (ftpm *FullTimeoutPhaseManager) OnPulse(ctx context.Context, pulse *core.Pulse) error {
	return nil
}

func (s *testSuite) TestFullTimeOut() {
	s.T().Skip("will be available after phase result fix !")
	networkNodesCount := 5
	phasesResult := make(chan error)
	bootstrapNode1 := s.createNetworkNode(s.T(), Disable)
	s.bootstrapNodes = append(s.bootstrapNodes, bootstrapNode1)

	s.testNode = s.createNetworkNode(s.T(), Full)

	for i := 0; i < networkNodesCount; i++ {
		s.networkNodes = append(s.networkNodes, s.createNetworkNode(s.T(), Disable))
	}

	s.InitNodes()
	s.StartNodes()
	res := <-phasesResult
	s.NoError(res)
	activeNodes := s.testNode.serviceNetwork.NodeKeeper.GetActiveNodes()
	s.Equal(1, len(activeNodes))
	// teardown
	<-time.After(time.Second * 5)
	s.StopNodes()
}

// Partitial timeout

func (s *testSuite) TestPartitionalTimeOut() {
	networkNodesCount := 5
	phasesResult := make(chan error)
	bootstrapNode1 := s.createNetworkNode(s.T(), Disable)
	s.bootstrapNodes = append(s.bootstrapNodes, bootstrapNode1)

	s.testNode = s.createNetworkNode(s.T(), Partitial)

	for i := 0; i < networkNodesCount; i++ {
		s.networkNodes = append(s.networkNodes, s.createNetworkNode(s.T(), Disable))
	}

	s.InitNodes()
	phase := &FirstPhase{
		NodeNetwork:  s.testNode.serviceNetwork.NodeNetwork,
		Calculator:   s.testNode.serviceNetwork.MerkleCalculator,
		Communicator: s.testNode.serviceNetwork.Communicator,
		Cryptography: s.testNode.serviceNetwork.CryptographyService,
		NodeKeeper:   s.testNode.serviceNetwork.NodeKeeper,
	}
	s.testNode.serviceNetwork.PhaseManager.(*PartitialTimeoutPhaseManager).FirstPhase = phase
	s.StartNodes()
	res := <-phasesResult
	s.NoError(res)
	// activeNodes := s.testNode.serviceNetwork.NodeKeeper.GetActiveNodes()
	// s.Equal(1, len(activeNodes))	// TODO: do test check
	// teardown
	<-time.After(time.Second * 5)
	s.StopNodes()
}

type FirstPhase struct {
	NodeNetwork  core.NodeNetwork         `inject:""`
	Calculator   merkle.Calculator        `inject:""`
	Communicator phases.Communicator      `inject:""`
	Cryptography core.CryptographyService `inject:""`
	NodeKeeper   network.NodeKeeper       `inject:""`
	UnsyncList   network.UnsyncList
}

func (fp *FirstPhase) Execute(ctx context.Context, pulse *core.Pulse) (*phases.FirstPhaseState, error) {
	entry := &merkle.PulseEntry{Pulse: pulse}
	pulseHash, pulseProof, err := fp.Calculator.GetPulseProof(entry)
	if fp.NodeKeeper.GetState() == network.Ready {
		fp.UnsyncList = fp.NodeKeeper.GetUnsyncList()
	}

	if err != nil {
		return nil, errors.Wrap(err, "[ Execute ] Failed to calculate pulse proof.")
	}

	packet := packets.Phase1Packet{}
	err = packet.SetPulseProof(pulseProof.StateHash, pulseProof.Signature.Bytes())
	if err != nil {
		return nil, errors.Wrap(err, "[ Execute ] Failed to set pulse proof in Phase1Packet.")
	}

	var success bool
	if fp.NodeKeeper.NodesJoinedDuringPreviousPulse() {
		originClaim, err := fp.NodeKeeper.GetOriginClaim()
		if err != nil {
			return nil, errors.Wrap(err, "[ Execute ] Failed to get origin claim")
		}
		success = packet.AddClaim(originClaim)
		if !success {
			return nil, errors.Wrap(err, "[ Execute ] Failed to add origin claim in Phase1Packet.")
		}
	}
	for {
		success = packet.AddClaim(fp.NodeKeeper.GetClaimQueue().Front())
		if !success {
			break
		}
		_ = fp.NodeKeeper.GetClaimQueue().Pop()
	}

	activeNodes := fp.NodeKeeper.GetActiveNodes()
	activeNodes = activeNodes[:len(activeNodes)-2] // delete 2 nodes
	err = fp.signPhase1Packet(&packet)
	if err != nil {
		return nil, errors.Wrap(err, "failed to sign a packet")
	}
	resultPackets, addressMap, err := fp.Communicator.ExchangePhase1(ctx, activeNodes, &packet)
	if err != nil {
		return nil, errors.Wrap(err, "[ Execute ] Failed to exchange results.")
	}

	proofSet := make(map[core.RecordRef]*merkle.PulseProof)
	claimMap := make(map[core.RecordRef][]packets.ReferendumClaim)
	for ref, packet := range resultPackets {
		signIsCorrect, err := fp.isSignPhase1PacketRight(packet, ref)
		if err != nil {
			log.Warn("failed to check a sign: ", err.Error())
		} else if !signIsCorrect {
			log.Warn("recieved a bad sign packet: ", err.Error())
		}
		rawProof := packet.GetPulseProof()
		proofSet[ref] = &merkle.PulseProof{
			BaseProof: merkle.BaseProof{
				Signature: core.SignatureFromBytes(rawProof.Signature()),
			},
			StateHash: rawProof.StateHash(),
		}
		claimMap[ref] = packet.GetClaims()
	}

	if fp.NodeKeeper.GetState() == network.Waiting {
		length, err := detectSparseBitsetLength(claimMap)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to detect bitset length")
		}
		fp.UnsyncList = fp.NodeKeeper.GetSparseUnsyncList(length)
	}

	fp.UnsyncList.AddClaims(claimMap, addressMap)
	valid, fault := fp.validateProofs(pulseHash, proofSet)

	return &phases.FirstPhaseState{
		PulseEntry:  entry,
		PulseHash:   pulseHash,
		PulseProof:  pulseProof,
		ValidProofs: valid,
		FaultProofs: fault,
		UnsyncList:  fp.UnsyncList,
	}, nil
}

type PartitialTimeoutPhaseManager struct {
	FirstPhase  *FirstPhase
	SecondPhase *phases.SecondPhase
	ThirdPhase  *phases.ThirdPhase
}

func (ftpm *PartitialTimeoutPhaseManager) OnPulse(ctx context.Context, pulse *core.Pulse) error {
	var err error

	pulseDuration, err := getPulseDuration(pulse)
	if err != nil {
		return errors.Wrap(err, "[ OnPulse ] Failed to get pulse duration")
	}

	var tctx context.Context
	var cancel context.CancelFunc

	tctx, cancel = contextTimeout(ctx, *pulseDuration, 0.2)
	defer cancel()

	firstPhaseState, err := ftpm.FirstPhase.Execute(tctx, pulse)

	if err != nil {
		return errors.Wrap(err, "[ TestCase.OnPulse ] failed to execute a phase")
	}

	tctx, cancel = contextTimeout(ctx, *pulseDuration, 0.2)
	defer cancel()

	secondPhaseState, err := ftpm.SecondPhase.Execute(tctx, firstPhaseState)
	checkError(err)

	fmt.Println(secondPhaseState) // TODO: remove after use
	checkError(ftpm.ThirdPhase.Execute(ctx, secondPhaseState))

	return nil
}

func contextTimeout(ctx context.Context, duration time.Duration, k float64) (context.Context, context.CancelFunc) {
	timeout := time.Duration(k * float64(duration))
	timedCtx, cancelFund := context.WithTimeout(ctx, timeout)
	return timedCtx, cancelFund
}

func getPulseDuration(pulse *core.Pulse) (*time.Duration, error) {
	duration := time.Duration(pulse.PulseNumber-pulse.PrevPulseNumber) * time.Second
	return &duration, nil
}

func (fp *FirstPhase) signPhase1Packet(packet *packets.Phase1Packet) error {
	data, err := packet.RawBytes()
	if err != nil {
		return errors.Wrap(err, "failed to get raw bytes")
	}
	sign, err := fp.Cryptography.Sign(data)
	if err != nil {
		return errors.Wrap(err, "failed to sign a phase 2 packet")
	}
	copy(packet.Signature[:], sign.Bytes())
	return nil
}

func (fp *FirstPhase) isSignPhase1PacketRight(packet *packets.Phase1Packet, recordRef core.RecordRef) (bool, error) {
	key := fp.NodeNetwork.GetActiveNode(recordRef).PublicKey()
	raw, err := packet.RawBytes()

	if err != nil {
		return false, errors.Wrap(err, "failed to serialize packet")
	}
	return fp.Cryptography.Verify(key, core.SignatureFromBytes(raw), raw), nil
}

func detectSparseBitsetLength(claims map[core.RecordRef][]packets.ReferendumClaim) (int, error) {
	// TODO: NETD18-47
	for _, claimList := range claims {
		for _, claim := range claimList {
			if claim.Type() == packets.TypeNodeAnnounceClaim {
				announceClaim, ok := claim.(*packets.NodeAnnounceClaim)
				if !ok {
					continue
				}
				return int(announceClaim.NodeCount), nil
			}
		}
	}
	return 0, errors.New("no announce claims were received")
}

func (fp *FirstPhase) validateProofs(
	pulseHash merkle.OriginHash,
	proofs map[core.RecordRef]*merkle.PulseProof,
) (valid map[core.Node]*merkle.PulseProof, fault map[core.RecordRef]*merkle.PulseProof) {

	validProofs := make(map[core.Node]*merkle.PulseProof)
	faultProofs := make(map[core.RecordRef]*merkle.PulseProof)
	for nodeID, proof := range proofs {
		valid := fp.validateProof(pulseHash, nodeID, proof)
		if valid {
			validProofs[fp.UnsyncList.GetActiveNode(nodeID)] = proof
		} else {
			faultProofs[nodeID] = proof
		}
	}
	return validProofs, faultProofs
}

func (fp *FirstPhase) validateProof(pulseHash merkle.OriginHash, nodeID core.RecordRef, proof *merkle.PulseProof) bool {
	node := fp.UnsyncList.GetActiveNode(nodeID)
	if node == nil {
		return false
	}
	return fp.Calculator.IsValid(proof, pulseHash, node.PublicKey())
}

func checkError(err error) {
	if err != nil {
		log.Error(err)
	}
}
