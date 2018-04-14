package signature

import (
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
)

// Signature interface of different signature algorithm
type Signature interface {

	// Algorithm returns the standard algorithm for this key.
	Algorithm() algorithm.Algorithm

	// InitSign this object for signing. If this method is called
	// again with a different argument, it negates the effect
	// of this call.
	InitSign(privateKey PrivateKey)

	// Sign returns the signature bytes of all the data input.
	// The format of the signature depends on the underlying
	// signature scheme.
	Sign(data []byte) (out []byte, err error)

	// RecoverPublic returns a public key, which is recovered by data and signature
	RecoverPublic(data []byte, signature []byte) (PublicKey, error)

	// InitVerify initializes this object for verification. If this method is called
	// again with a different argument, it negates the effect
	// of this call.
	InitVerify(publicKey PublicKey)

	// Verify the passed-in signature.
	//
	// <p>A call to this method resets this signature object to the state
	// it was in when previously initialized for verification via a
	// call to <code>initVerify(PublicKey)</code>. That is, the object is
	// reset and available to verify another signature from the identity
	// whose public key was specified in the call to <code>initVerify</code>.
	Verify(data []byte, signature []byte) (bool, error)
}
