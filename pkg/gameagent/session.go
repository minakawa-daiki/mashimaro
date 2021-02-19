package gameagent

import (
	"context"
	"log"
	"time"

	"github.com/castaneai/mashimaro/pkg/gamemetadata"
	"github.com/castaneai/mashimaro/pkg/proto"
	"github.com/goccy/go-yaml"
	"github.com/pkg/errors"
)

func waitForSession(ctx context.Context, c proto.BrokerClient, gsName string) (*proto.Session, error) {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			resp, err := c.FindSession(ctx, &proto.FindSessionRequest{GameserverName: gsName})
			if err != nil {
				return nil, errors.Wrap(err, "failed to find session by broker")
			}
			if !resp.Found {
				continue
			}
			var metadata gamemetadata.Metadata
			if err := yaml.Unmarshal([]byte(resp.Session.GameMetadata.Body), &metadata); err != nil {
				return nil, err
			}
			log.Printf("found session(sid: %s, gameId: %s)", resp.Session.SessionId, metadata.GameID)
			return resp.Session, nil
		}
	}
}
