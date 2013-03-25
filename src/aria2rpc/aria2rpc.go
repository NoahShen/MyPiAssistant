package aria2rpc

import (
	"github.com/kdar/httprpc"
)

const (
	RPC_VERSION = "2.0"
	RPC_URL     = "http://127.0.0.1:6800/jsonrpc"
)

func AddUri(uri string, params map[string]string) (string, error) {
	method := "aria2.addUri"
	paramArr := make([]interface{}, 1)
	uris := make([]string, 1)
	uris[0] = uri
	paramArr[0] = uris
	if params != nil {
		paramArr = append(paramArr, params)
	}

	var replyGID string
	err := httprpc.CallJson(RPC_VERSION, RPC_URL, method, paramArr, &replyGID)
	if err != nil {
		return "", err
	}
	return replyGID, nil
}

func GetActive(keys []string) ([]map[string]interface{}, error) {
	method := "aria2.tellActive"
	paramArr := make([]interface{}, 0)
	if keys != nil && len(keys) > 0 {
		paramArr = append(paramArr, keys)
	}
	var reply = make([]map[string]interface{}, 10)
	err := httprpc.CallJson(RPC_VERSION, RPC_URL, method, paramArr, &reply)
	if err != nil {
		return nil, err
	}
	return reply, nil
}
