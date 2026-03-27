package ogmios

import (
	"encoding/json"

	"github.com/Ethernal-Tech/cardano-infrastructure/indexer"
	"github.com/gorilla/websocket"
)

func sendFindIntersection(conn *websocket.Conn, bp indexer.BlockPoint) error {
	if bp.BlockSlot == 0 {
		return sendRPC(
			conn,
			findIntersectionMethod,
			ogmiosIntersection[string]{
				Points: []string{"origin"},
			},
			findIntersectionID)
	}

	return sendRPC(
		conn,
		findIntersectionMethod,
		ogmiosIntersection[ogmiosPoint]{
			Points: []ogmiosPoint{newOgmiosPoint(bp)},
		},
		findIntersectionID)
}

func sendNextBlock(conn *websocket.Conn) error {
	return sendRPC(
		conn,
		nextBlockMethod,
		struct{}{},
		nextBlockID)
}

func sendRPC(conn *websocket.Conn, method string, params any, id string) error {
	data, err := json.Marshal(ogmiosRequest{
		Version: "2.0",
		Method:  method,
		Params:  params,
		ID:      id,
	})
	if err != nil {
		return err
	}

	return conn.WriteMessage(websocket.TextMessage, data)
}
