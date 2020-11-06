package solana

import (
	"fmt"
	"net/rpc"
)

type RPCClient struct {
	client             *rpc.Client
	solanaEndpointAddr string
}

func NewRPCClient(endpointAddr string) *RPCClient {
	return &RPCClient{
		solanaEndpointAddr: endpointAddr,
	}
}

func (r *RPCClient) getClient() (*rpc.Client, error) {

	if r.client == nil {
		client, err := rpc.Dial("tcp", r.solanaEndpointAddr)
		if err != nil {
			return nil, fmt.Errorf("client tcp dialing to address %s: %w", r.solanaEndpointAddr, err)
		}
		r.client = client
	}

	return r.client, nil
}

func (r *RPCClient) GetConfirmedBlock(slot uint64) error {

	c, err := r.getClient()
	if err != nil {
		return err
	}

	var reply struct {
		ID int
	}

	err = c.Call("getConfirmedBlock", &struct {
		Slot     uint64 `json:"slot"`
		Encoding string `json:"encoding"`
	}{
		Slot:     slot,
		Encoding: "json",
	}, &reply)

	if err != nil {
		return fmt.Errorf("calling getConfirmedBlock: %w", err)
	}

	return nil
}
