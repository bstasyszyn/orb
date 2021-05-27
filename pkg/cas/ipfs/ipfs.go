/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ipfs

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	shell "github.com/ipfs/go-ipfs-api"
	log "github.com/sirupsen/logrus"

	"github.com/trustbloc/orb/pkg/store/cas"
)

const timeout = 2

// Client will write new documents to IPFS and read existing documents from IPFS based on CID.
// It implements Sidetree CAS interface.
type Client struct {
	ipfs      *shell.Shell
	useV0CIDs bool
}

// New creates cas client.
func New(url string, useV0CIDs bool) *Client {
	ipfs := shell.NewShell(url)

	ipfs.SetTimeout(timeout * time.Second)

	return &Client{ipfs: ipfs, useV0CIDs: useV0CIDs}
}

// Write writes the given content to CAS.
// returns cid which represents the address of the content.
func (m *Client) Write(content []byte) (string, error) {
	var v1AddOpt []shell.AddOpts

	if !m.useV0CIDs {
		v1AddOpt = []shell.AddOpts{shell.CidVersion(1)}
	}

	cid, err := m.ipfs.Add(bytes.NewReader(content), v1AddOpt...)
	if err != nil {
		return "", err
	}

	log.Debugf("added content returned cid: %s", cid)

	return cid, nil
}

// Read reads the content for the given CID from CAS.
// returns the contents of CID.
func (m *Client) Read(cid string) ([]byte, error) {
	reader, err := m.ipfs.Cat(cid)
	if err != nil {
		if strings.Contains(err.Error(), "context deadline exceeded") {
			return nil, fmt.Errorf("%s: %w", err.Error(), cas.ErrContentNotFound)
		}

		return nil, err
	}

	defer closeAndLog(reader)

	return ioutil.ReadAll(reader)
}

func closeAndLog(rc io.Closer) {
	if err := rc.Close(); err != nil {
		log.Warnf("failed to close reader: %s", err.Error())
	}
}