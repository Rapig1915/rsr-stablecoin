// +build all

package tests

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"

	"github.com/reserve-protocol/rsv-beta/abi"
)

func TestOwnable(t *testing.T) {
	suite.Run(t, new(OwnableSuite))
}

type OwnableSuite struct {
	TestSuite

	ownable        *abi.BasicOwnable
	ownableAddress common.Address
}

var (
	// Compile-time check that OwnableSuite implements the interfaces we think it does.
	// If it does not implement these interfaces, then the corresponding setup and teardown
	// functions will not actually run.
	_ suite.BeforeTest       = &OwnableSuite{}
	_ suite.SetupAllSuite    = &OwnableSuite{}
	_ suite.TearDownAllSuite = &OwnableSuite{}
)

// SetupSuite runs once, before all of the tests in the suite.
func (s *OwnableSuite) SetupSuite() {
	s.setup()
}

// BeforeTest runs before each test in the suite.
func (s *OwnableSuite) BeforeTest(suiteName, testName string) {
	s.owner = s.account[0]

	// Deploy BasicOwnable.
	ownableAddress, tx, ownable, err := abi.DeployBasicOwnable(s.signer, s.node)

	s.logParsers = map[common.Address]logParser{
		ownableAddress: ownable,
	}
	s.ownable = ownable
	s.ownableAddress = ownableAddress

	s.requireTxWithStrictEvents(tx, err)(
		abi.BasicOwnableOwnershipTransferred{
			PreviousOwner: zeroAddress(), NewOwner: s.owner.address(),
		},
	)
}

func (s *OwnableSuite) TestDeploy() {}

// TestConstructor tests that the constructor sets initial state appropriately.
func (s *OwnableSuite) TestConstructor() {
	// Initial owner should be deployer.
	ownerAddress, err := s.ownable.Owner(nil)
	s.Require().NoError(err)
	s.Equal(s.owner.address(), ownerAddress)

	// Initial nominated owner should be the zero address.
	nominatedOwnerAddress, err := s.ownable.NominatedOwner(nil)
	s.Require().NoError(err)
	s.Equal(zeroAddress(), nominatedOwnerAddress)
}

// TestNominateNewOwner unit tests the nominateNewOwner function.
func (s *OwnableSuite) TestNominateNewOwner() {
	newOwner := s.account[1]
	s.requireTxWithStrictEvents(s.ownable.NominateNewOwner(s.signer, newOwner.address()))(
		abi.BasicOwnableNewOwnerNominated{
			PreviousOwner: s.owner.address(), Nominee: newOwner.address(),
		},
	)

	// Check that state changed appropriately.
	nominatedOwnerAddress, err := s.ownable.NominatedOwner(nil)
	s.Require().NoError(err)
	s.Equal(newOwner.address(), nominatedOwnerAddress)
}

// TestNominateNewOwnerNegativeCases makes sure nominateNewOwner reverts when it is supposed to.
func (s *OwnableSuite) TestNominateNewOwnerNegativeCases() {
	newOwner := s.account[1]
	s.requireTxFails(s.ownable.NominateNewOwner(s.signer, zeroAddress()))
	s.requireTxFails(s.ownable.NominateNewOwner(signer(newOwner), newOwner.address()))

	// Check that the nominated owner cannot call nominateNewOwner.
	s.requireTxWithStrictEvents(s.ownable.NominateNewOwner(s.signer, newOwner.address()))(
		abi.BasicOwnableNewOwnerNominated{
			PreviousOwner: s.owner.address(), Nominee: newOwner.address(),
		},
	)

	s.requireTxFails(s.ownable.NominateNewOwner(signer(newOwner), s.account[2].address()))
}

// TestAcceptOwnershipByNominatedOwner tests that ownership can be accepted by nominated owner.
func (s *OwnableSuite) TestAcceptOwnershipByNominatedOwner() {
	newOwner := s.account[1]
	s.requireTxWithStrictEvents(s.ownable.NominateNewOwner(s.signer, newOwner.address()))(
		abi.BasicOwnableNewOwnerNominated{
			PreviousOwner: s.owner.address(), Nominee: newOwner.address(),
		},
	)

	// Check that the nominated owner can accept ownership.
	s.requireTxWithStrictEvents(s.ownable.AcceptOwnership(signer(newOwner)))(
		abi.BasicOwnableOwnershipTransferred{
			PreviousOwner: s.owner.address(), NewOwner: newOwner.address(),
		},
	)

	// Check that state changed appropriately.
	ownerAddress, err := s.ownable.Owner(nil)
	s.Require().NoError(err)
	s.Equal(ownerAddress, newOwner.address())
}

// TestAcceptOwnershipNegativeCases makes sure acceptOwner reverts when it is supposed to.
func (s *OwnableSuite) TestAcceptOwnershipNegativeCases() {
	newOwner := s.account[1]

	// Check that acceptOwnership cannot be used to make owner the zero address.
	s.requireTxFails(s.ownable.AcceptOwnership(s.signer))

	// Set nominatedOwner.
	s.requireTxWithStrictEvents(s.ownable.NominateNewOwner(s.signer, newOwner.address()))(
		abi.BasicOwnableNewOwnerNominated{
			PreviousOwner: s.owner.address(), Nominee: newOwner.address(),
		},
	)

	// Check that a random address cannot accept ownership for the nominatedOwner.
	s.requireTxFails(s.ownable.AcceptOwnership(signer(s.account[2])))

	// Check that the current owner cannot force ownership onto the nominatedOwner.
	s.requireTxFails(s.ownable.AcceptOwnership(s.signer))
}

// TestRenounceOwnership unit tests the renounceOwnership function.
func (s *OwnableSuite) TestRenounceOwnership() {
	// Check that the owner can renounce ownership.
	pledge := "I hereby renounce ownership of this contract forever."
	s.requireTxWithStrictEvents(s.ownable.RenounceOwnership(s.signer, pledge))(
		abi.BasicOwnableOwnershipTransferred{
			PreviousOwner: s.owner.address(), NewOwner: zeroAddress(),
		},
	)

	// Check that state changed appropriately.
	ownerAddress, err := s.ownable.Owner(nil)
	s.Require().NoError(err)
	s.Equal(ownerAddress, zeroAddress())
}

// TestRenounceOwnershipNegativeCases makes sure renounceOwnership can only be called by owner.
func (s *OwnableSuite) TestRenounceOwnershipNegativeCases() {
	pledge := "I hereby renounce ownership of this contract forever."
	s.requireTxFails(s.ownable.RenounceOwnership(signer(s.account[1]), pledge))
	s.requireTxFails(s.ownable.RenounceOwnership(s.signer, "mumble frotz"))

	// Check that the nominated owner cannot call nominateNewOwner.
	newOwner := s.account[1]
	s.requireTxWithStrictEvents(s.ownable.NominateNewOwner(s.signer, newOwner.address()))(
		abi.BasicOwnableNewOwnerNominated{
			PreviousOwner: s.owner.address(), Nominee: newOwner.address(),
		},
	)
	s.requireTxFails(s.ownable.RenounceOwnership(signer(newOwner), pledge))
}

// TestUseCases chains a bunch of calls into each other in a more realistic test of things.
func (s *OwnableSuite) TestUseCases() {
	firstOwner := s.account[1]
	secondOwner := s.account[2]

	// Nominate the first owner.
	s.requireTxWithStrictEvents(s.ownable.NominateNewOwner(s.signer, firstOwner.address()))(
		abi.BasicOwnableNewOwnerNominated{
			PreviousOwner: s.owner.address(), Nominee: firstOwner.address(),
		},
	)

	// Should not be able to accept as anyone else.
	s.requireTxFails(s.ownable.AcceptOwnership(s.signer))
	s.requireTxFails(s.ownable.AcceptOwnership(signer(secondOwner)))

	// Check that the nominated owner can accept ownership.
	s.requireTxWithStrictEvents(s.ownable.AcceptOwnership(signer(firstOwner)))(
		abi.BasicOwnableOwnershipTransferred{
			PreviousOwner: s.owner.address(), NewOwner: firstOwner.address(),
		},
	)

	// Should not be able to accept as anyone else after either.
	s.requireTxFails(s.ownable.AcceptOwnership(s.signer))
	s.requireTxFails(s.ownable.AcceptOwnership(signer(secondOwner)))

	// Even the original owner shouldn't be able to call nominate again, especially on themselves.
	s.requireTxFails(s.ownable.NominateNewOwner(s.signer, s.owner.address()))
	s.requireTxFails(s.ownable.NominateNewOwner(s.signer, secondOwner.address()))

	// Nominate the second owner.
	s.requireTxWithStrictEvents(s.ownable.NominateNewOwner(signer(firstOwner), secondOwner.address()))(
		abi.BasicOwnableNewOwnerNominated{
			PreviousOwner: firstOwner.address(), Nominee: secondOwner.address(),
		},
	)

	// Should not be able to accept as anyone else.
	s.requireTxFails(s.ownable.AcceptOwnership(s.signer))
	s.requireTxFails(s.ownable.AcceptOwnership(signer(firstOwner)))

	// Check that the nominated owner can accept ownership.
	s.requireTxWithStrictEvents(s.ownable.AcceptOwnership(signer(secondOwner)))(
		abi.BasicOwnableOwnershipTransferred{
			PreviousOwner: firstOwner.address(), NewOwner: secondOwner.address(),
		},
	)

	// Should not be able to accept as anyone else after either.
	s.requireTxFails(s.ownable.AcceptOwnership(s.signer))
	s.requireTxFails(s.ownable.AcceptOwnership(signer(firstOwner)))

	// Even the original owner shouldn't be able to call nominate again, especially on themselves.
	s.requireTxFails(s.ownable.NominateNewOwner(s.signer, s.owner.address()))
	s.requireTxFails(s.ownable.NominateNewOwner(s.signer, firstOwner.address()))

	// And calling AcceptOwnership again shouldn't matter, but should emit a weird looking event.
	s.requireTxWithStrictEvents(s.ownable.AcceptOwnership(signer(secondOwner)))(
		abi.BasicOwnableOwnershipTransferred{
			PreviousOwner: secondOwner.address(), NewOwner: secondOwner.address(),
		},
	)

	// Should not be able to accept as anyone else after either.
	s.requireTxFails(s.ownable.AcceptOwnership(s.signer))
	s.requireTxFails(s.ownable.AcceptOwnership(signer(firstOwner)))

	// Even the original owner shouldn't be able to call nominate again, especially on themselves.
	s.requireTxFails(s.ownable.NominateNewOwner(s.signer, s.owner.address()))
	s.requireTxFails(s.ownable.NominateNewOwner(signer(firstOwner), firstOwner.address()))
}
