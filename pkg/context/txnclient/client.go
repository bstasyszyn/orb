/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package txnclient

import (
	"sync"

	"github.com/trustbloc/sidetree-core-go/pkg/api/operation"
	txnapi "github.com/trustbloc/sidetree-core-go/pkg/api/txn"

	"github.com/trustbloc/orb/pkg/didtxnref"
	"github.com/trustbloc/orb/pkg/txngraph"
)

// Client implements writing orb transactions.
type Client struct {
	namespace string
	txnGraph  txnGraph
	didTxns   didTxns
	txnCh     chan []string
	sync.RWMutex
}

type txnGraph interface {
	Add(info *txngraph.Node) (string, error)
}

type didTxns interface {
	Add(did, cid string) error
	Get(did string) ([]string, error)
}

// New returns a new orb transaction client.
func New(namespace string, graph txnGraph, txns didTxns, txnCh chan []string) *Client {
	return &Client{
		namespace: namespace,
		txnGraph:  graph,
		didTxns:   txns,
		txnCh:     txnCh,
	}
}

// WriteAnchor writes anchor string to orb transaction.
func (c *Client) WriteAnchor(anchor string, refs []*operation.Reference, version uint64) error {
	// assemble map of previous did transaction for each did that is referenced in anchor
	previousDidTxns := make(map[string]string)

	for _, ref := range refs {
		txns, err := c.didTxns.Get(ref.UniqueSuffix)
		if err != nil && err != didtxnref.ErrDidTransactionReferencesNotFound {
			return err
		}

		// TODO: it is ok for transaction references not to be there for create; handle other types here

		// get did's last transaction
		if len(txns) > 0 {
			previousDidTxns[ref.UniqueSuffix] = txns[len(txns)-1]
		}
	}

	txnInfo := &txngraph.Node{
		AnchorString:   anchor,
		Namespace:      c.namespace,
		Version:        version,
		PreviousDidTxn: previousDidTxns,
	}

	cid, err := c.txnGraph.Add(txnInfo)
	if err != nil {
		return err
	}

	// update global did/txn references
	for _, ref := range refs {
		addErr := c.didTxns.Add(ref.UniqueSuffix, cid)
		if addErr != nil {
			return addErr
		}
	}

	c.txnCh <- []string{cid}

	return nil
}

// Read reads transactions since transaction time.
// TODO: This is not used and can be removed from interface if we change observer in sidetree-mock to point
// to core observer (can be done easily) Concern: Reference app has this interface.
func (c *Client) Read(_ int) (bool, *txnapi.SidetreeTxn) {
	// not used
	return false, nil
}
